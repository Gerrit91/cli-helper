package battery

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

type (
	daemon struct {
		log       *slog.Logger
		daemonize bool
		mu        sync.Mutex

		sessionBus *dbus.Conn

		hasEmittedLow           bool
		hasEmittedCriticallyLow bool
		hasEmittedCharging      bool
		hasEmittedDischarging   bool
		hasEmittedFullyCharged  bool
	}

	notification struct {
		message string
		urgency urgency
		icon    icon
		timeout *time.Duration
	}

	urgency string
	icon    string
)

const (
	appName           = "battery-daemon"
	notificationSound = "/usr/share/sounds/freedesktop/stereo/message.oga"

	criticallyLowPercentage = 10
	lowPercentage           = 20
	fullPercentage          = 90

	urgencyLow      urgency = "low"
	urgencyNormal   urgency = "normal"
	urgencyCritical urgency = "critical"

	iconFull     icon = "battery-full"
	iconEmpty    icon = "battery-caution"
	iconLow      icon = "battery-low"
	iconCharging icon = "battery-full-charging"
)

func New(daemonize bool) *daemon {
	return &daemon{
		log:       slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
		daemonize: daemonize,
	}
}

func (d *daemon) Run(ctx context.Context) error {
	d.log.Info("battery daemon starting...")

	sessionBus, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("dbus: failed to get session bus: %w", err)
	}

	d.sessionBus = sessionBus

	percentage, isCharging, err := getInitialState()
	if err != nil {
		return fmt.Errorf("unable to get initial battery state: %w", err)
	}

	d.log.Info("emitting startup notifications if necessary", "percentage", percentage, "charging", isCharging)

	d.evaluateAndEmit(percentage, isCharging)

	if !d.daemonize {
		return nil
	}

	systemBus, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("dbus: failed to get system bus: %w", err)
	}

	rule := "type='signal',sender='org.freedesktop.UPower',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged'"

	err = systemBus.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule).Err
	if err != nil {
		return fmt.Errorf("failed to add dbus match rule: %w", err)
	}

	err = systemBus.AddMatchSignal(
		dbus.WithMatchSender("org.freedesktop.UPower"),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
	)
	if err != nil {
		return fmt.Errorf("failed to add dbus match signal: %w", err)
	}

	var (
		debounceTimer    *time.Timer
		c                = make(chan *dbus.Signal, 10)
		evaluateTrigger  = make(chan struct{}, 1)
		debounceDuration = 400 * time.Millisecond
	)

	systemBus.Signal(c)

	d.log.Info("listening for system bus events...")

	for {
		select {
		case <-evaluateTrigger:
			obj := systemBus.Object("org.freedesktop.UPower", dbus.ObjectPath("/org/freedesktop/UPower/devices/battery_BAT0"))

			percentageVal, err := obj.GetProperty("org.freedesktop.UPower.Device.Percentage")
			if err != nil {
				d.log.Error("percentage property missing", "error", err)
				continue
			}
			stateVal, err := obj.GetProperty("org.freedesktop.UPower.Device.State")
			if err != nil {
				d.log.Error("state property missing", "error", err)
				continue
			}

			var (
				percentage = int(percentageVal.Value().(float64))
				isCharging bool
			)

			switch stateVal.Value().(uint32) {
			case 1:
				isCharging = true
			default:
				isCharging = false
			}

			d.log.Info("evaluating state", "percentage", percentage, "charging", isCharging)

			d.evaluateAndEmit(percentage, isCharging)

		case <-c:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(debounceDuration, func() {
				evaluateTrigger <- struct{}{}
			})

		case <-ctx.Done():
			d.log.Info("shutting down battery daemon")
			return nil
		}
	}
}

func (d *daemon) evaluateAndEmit(percentage int, isCharging bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if isCharging {
		if !d.hasEmittedCharging {
			d.emitNotification(&notification{
				message: fmt.Sprintf("Battery is charging (%d%%)", percentage),
				urgency: urgencyLow,
				icon:    iconCharging,
				timeout: new(10 * time.Second),
			})
			d.emitSound()
			defer func() {
				d.hasEmittedCharging = true
			}()
		}

		d.hasEmittedDischarging = false
	} else {
		if d.hasEmittedCharging && !d.hasEmittedDischarging {
			d.emitNotification(&notification{
				message: fmt.Sprintf("Stopped charging (%d%%)", percentage),
				urgency: urgencyLow,
				icon:    iconFull,
				timeout: new(10 * time.Second),
			})
			d.emitSound()
			d.hasEmittedDischarging = true
		}

		d.hasEmittedCharging = false
	}

	if percentage > criticallyLowPercentage {
		d.hasEmittedCriticallyLow = false
	}
	if percentage > lowPercentage {
		d.hasEmittedLow = false
	}
	if percentage < fullPercentage {
		d.hasEmittedFullyCharged = false
	}

	if isCharging {
		if percentage >= fullPercentage {
			if d.hasEmittedCharging && !d.hasEmittedFullyCharged {
				d.emitNotification(&notification{
					message: fmt.Sprintf("Battery is sufficiently charged (%d%%)", percentage),
					urgency: urgencyNormal,
					icon:    iconFull,
					timeout: new(5 * time.Second),
				})
				d.hasEmittedFullyCharged = true
			}
		}

		return
	}

	if percentage <= criticallyLowPercentage {
		if !d.hasEmittedCriticallyLow {
			d.emitNotification(&notification{
				message: fmt.Sprintf("Battery is critically low (%d%%)", percentage),
				urgency: urgencyCritical,
				icon:    iconEmpty,
			})
			d.emitSound()
			d.hasEmittedCriticallyLow = true
		}
	} else if percentage <= lowPercentage {
		if !d.hasEmittedLow {
			d.emitNotification(&notification{
				message: fmt.Sprintf("Battery is running low (%d%%)", percentage),
				urgency: urgencyNormal,
				icon:    iconLow,
			})
			d.emitSound()
			d.hasEmittedLow = true
		}
	}
}

func (d *daemon) emitNotification(n *notification) {
	const (
		title = "Battery Status"
	)

	d.log.Info("sending dbus notification", "urgency", n.urgency, "icon", n.icon, "title", title, "message", n.message)

	var (
		notificationID uint32
		urgency        uint8
		timeout        = -1 // don't expire
	)

	switch n.urgency {
	case urgencyCritical:
		urgency = 2
	case urgencyNormal:
		urgency = 1
	case urgencyLow:
		urgency = 0
	default:
		urgency = 0
	}

	if n.timeout != nil {
		timeout = int(n.timeout.Milliseconds()) //
	}

	err := d.sessionBus.Object(
		"org.freedesktop.Notifications", dbus.ObjectPath("/org/freedesktop/Notifications")).
		Call("org.freedesktop.Notifications.Notify",
			0,
			appName,
			uint32(0), // replaces_id
			string(n.icon),
			title,
			n.message,
			[]string{}, // actions
			map[string]dbus.Variant{
				"urgency":    dbus.MakeVariant(byte(urgency)),
				"image-path": dbus.MakeVariant(string(n.icon)),
			}, // hints
			timeout,
		).Store(&notificationID)

	if err != nil {
		d.log.Error("failed to send notification", "error", err)
	}

	_ = notificationID
}

func (d *daemon) emitSound() {
	if err := exec.Command("mpv", notificationSound).Run(); err != nil {
		d.log.Error("failed to play sound", "error", err)
	}
}

func getInitialState() (int, bool, error) {
	capacityRaw, err := os.ReadFile("/sys/class/power_supply/BAT0/capacity")
	if err != nil {
		return 0, false, fmt.Errorf("failed to read battery capacity: %w", err)
	}

	capacity, err := strconv.Atoi(strings.TrimSpace(string(capacityRaw)))
	if err != nil {
		return 0, false, fmt.Errorf("failed to parse battery capacity: %w", err)
	}

	isCharging := false

	status, err := os.ReadFile("/sys/class/power_supply/BAT0/status")
	if err != nil {
		return 0, false, fmt.Errorf("failed to parse battery status: %w", err)
	}

	switch strings.TrimSpace(string(status)) {
	case "Charging", "Full":
		isCharging = true
	}

	return capacity, isCharging, nil
}
