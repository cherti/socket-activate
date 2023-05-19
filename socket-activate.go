package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/godbus/dbus"
)

var (
	mode               = flag.String("m", "tcp", "mode, available: tcp")
	targetUnit         = flag.String("u", "null.service", "corresponding unit")
	destinationAddress = flag.String("a", "127.0.0.1:80", "destination address")
	timeout            = flag.Duration("t", 0, "inactivity timeout after which to stop the unit again")
	retries            = flag.Uint("r", 10, "number of connection attempts (with 100ms delay) before giving up")
)

type unitController struct {
	conn     *dbus.Conn
	unitname string
}

func newUnitController(name string) unitController {
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Fatal(err)
	}
	return unitController{conn, name}
}

func (unitCtrl unitController) startSystemdUnit() {
	var responseObjPath dbus.ObjectPath
	obj := unitCtrl.conn.Object("org.freedesktop.systemd1", dbus.ObjectPath("/org/freedesktop/systemd1"))
	err := obj.Call("org.freedesktop.systemd1.Manager.StartUnit", 0, unitCtrl.unitname, "replace").Store(&responseObjPath)
	if err != nil {
		log.Fatal(err)
	}

}

func (unitCtrl unitController) stopSystemdUnit() {
	var responseObjPath dbus.ObjectPath
	obj := unitCtrl.conn.Object("org.freedesktop.systemd1", dbus.ObjectPath("/org/freedesktop/systemd1"))
	err := obj.Call("org.freedesktop.systemd1.Manager.StopUnit", 0, unitCtrl.unitname, "replace").Store(&responseObjPath)
	if err != nil {
		log.Fatal(err)
	}

}

func (unitCtrl unitController) terminateWithoutActivity(activity <-chan bool) {
	var timeoutCh <-chan time.Time
	if *timeout > 0 {
		timeoutCh = time.After(*timeout)
	}

	for {
		select {
		case <-activity:
		case <-timeoutCh:
			unitCtrl.stopSystemdUnit()
			os.Exit(0)
		}
	}
}

func proxyNetworkConnections(from net.Conn, to net.Conn, activityMonitor chan<- bool) {
	buffer := make([]byte, 1024)

	for {
		i, err := from.Read(buffer)
		if err != nil {
			from.Close()
			to.Close()
			return // EOF (if anything else, we scrap the connection anyways)
		}
		activityMonitor <- true
		to.Write(buffer[:i])
	}
}

func startTCPProxy(activityMonitor chan<- bool) {
	l, err := net.FileListener(os.NewFile(3, "systemd-socket"))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		activityMonitor <- true
		connOutwards, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		var connBackend net.Conn
		var tryCount uint
		for tryCount = 0; tryCount < *retries; tryCount++ {
			connBackend, err = net.Dial("tcp", *destinationAddress)
			if err != nil {
				fmt.Println(err)
				time.Sleep(100 * time.Millisecond)
				continue
			} else {
				break
			}
		}
		if tryCount >= *retries {
			continue
		}

		go proxyNetworkConnections(connOutwards, connBackend, activityMonitor)
		go proxyNetworkConnections(connBackend, connOutwards, activityMonitor)
	}
}

func main() {

	flag.Parse()

	if os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()) {

		unitCtrl := newUnitController(*targetUnit)

		activityMonitor := make(chan bool)
		go unitCtrl.terminateWithoutActivity(activityMonitor)

		// first, connect to systemd for starting the unit
		unitCtrl.startSystemdUnit()

		// then take over the socket from systemd
		startTCPProxy(activityMonitor)
	} else {
		log.Fatal("seems not to be systemd-activated, aborting")
	}
}
