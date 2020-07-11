// +build windows

package proc

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	win "github.com/eycorsican/go-tun2socks/common/winsys"
)

const (
	IdleProcessID   = 0
	SystemProcessID = 4
)

func GetPidBySocket(network string, addr string, port uint16) (int, error) {
	switch network {
	case "tcp":
		tcpTable, err := getTcp4Table()
		if err != nil {
			return 0, fmt.Errorf("failed to get TCP table: %v", err)
		}
		for i := 0; i < int(tcpTable.NumEntries); i++ {
			row := tcpTable.Table[i]
			if win.NTOHS(uint16(row.LocalPort)) == port /* && win.IPAddrNTOA(uint32(row.LocalAddr)) == addr */ {
				return int(row.OwningPid), nil
			}
		}
		return 0, errors.New("not found")
	case "udp":
		udpTable, err := getUdp4Table()
		if err != nil {
			return 0, fmt.Errorf("failed to get UDP table: %v", err)
		}
		for i := 0; i < int(udpTable.NumEntries); i++ {
			row := udpTable.Table[i]
			if win.NTOHS(uint16(row.LocalPort)) == port /* && win.IPAddrNTOA(uint32(row.LocalAddr)) == addr */ {
				return int(row.OwningPid), nil
			}
		}
		return 0, errors.New("not found")
	default:
		return 0, errors.New("not found")
	}
}

func GetProcessesBySocket(network string, addr string, port uint16) ([]string, error) {
	var processes []string
	var err error
	pid, err := GetPidBySocket(network, addr, port)
	if err != nil {
		return nil, err
	}
	for {
		ppid, comm, err := GetPpidAndCommand(pid)
		if err != nil {
			break // no more parents
		}
		fmt.Println("GetProcessBySocket result ", ppid, " ", comm," ")
		processes = append(processes, comm)
		if ppid == SystemProcessID || ppid == IdleProcessID {
			break
		}
		pid = ppid
	}
	if len(processes) == 0 {
		return nil, fmt.Errorf("not found")
	}
	return processes, nil
}

func GetCommandNameBySocket(network string, addr string, port uint16) (string, error) {
	switch network {
	case "tcp":
		tcpTable, err := getTcp4Table()
		if err != nil {
			return "", fmt.Errorf("failed to get TCP table: %v", err)
		}
		for i := 0; i < int(tcpTable.NumEntries); i++ {
			row := tcpTable.Table[i]
			if win.NTOHS(uint16(row.LocalPort)) == port /* && win.IPAddrNTOA(uint32(row.LocalAddr)) == addr */ {
				return getNameByPid(uint32(row.OwningPid))
			}
		}
		return "", errors.New("not found")
	case "udp":
		udpTable, err := getUdp4Table()
		if err != nil {
			return "", fmt.Errorf("failed to get UDP table: %v", err)
		}
		for i := 0; i < int(udpTable.NumEntries); i++ {
			row := udpTable.Table[i]
			if win.NTOHS(uint16(row.LocalPort)) == port /* && win.IPAddrNTOA(uint32(row.LocalAddr)) == addr */ {
				return getNameByPid(uint32(row.OwningPid))
			}
		}
		return "", errors.New("not found")
	default:
		return "", errors.New("not found")
	}
}

func GetPpidAndCommand(pid int) (int, string, error) {
	handle, err := syscall.CreateToolhelp32Snapshot(
		syscall.TH32CS_SNAPPROCESS,
		// This argument will be ignored with a SNAPPROCESS flag.
		// The snapshot will contain all processes, we will need
		// to iterate over them and filter the desired one.
		0,
	)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create snapshot: %v", err)
	}
	defer syscall.CloseHandle(handle)

	var ppid int
    var cmd2 string

	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	err = syscall.Process32First(handle, &pe)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get process entry: %v", err)
	}

	if int(pe.ProcessID) == pid {
		ppid = int(pe.ParentProcessID)
		cmd2  = syscall.UTF16ToString(pe.ExeFile[:])
		return  ppid,cmd2,nil
	} else {
		for {
			var pe syscall.ProcessEntry32
			pe.Size = uint32(unsafe.Sizeof(pe))
			err = syscall.Process32Next(handle, &pe)
			if err != nil {
				return 0, "", fmt.Errorf("failed to get next process entry: %v", err)
			}
		//	fmt.Println(pe.ProcessID, "  ",syscall.UTF16ToString(pe.ExeFile[:]))
			if int(pe.ProcessID) == pid {
				ppid = int(pe.ParentProcessID)
				cmd2  = syscall.UTF16ToString(pe.ExeFile[:])
				fmt.Println("now  ppid ",ppid, " cmd2 ", cmd2)
				return ppid,cmd2,nil
			}
		}
	}

	//mhandle, err := syscall.CreateToolhelp32Snapshot(
	//	syscall.TH32CS_SNAPMODULE,
	//	uint32(pid), // Create a snapshot only related to the specific pid.
	//)
	//if err != nil {
	//	return 0, "", fmt.Errorf("failed to create snapshot: %v", err)
	//}
	//defer syscall.CloseHandle(mhandle)

	//var me win.ModuleEntry32
	//me.Size = uint32(unsafe.Sizeof(me))
	//err = win.Module32First(win.Handle(mhandle), &me)
	//if err != nil {
	//	return 0, "", fmt.Errorf("failed to get module entry: %v", err)
	//}
	//
	//cmd := win.UTF16PtrToString(&me.Module[0])

	//cmd :=syscall.UTF16ToString(pe.ExeFile[:])
   // fmt.Println("pid is ", pid," ppid is ", ppid, " process name is ", cmd2)

	//return ppid, cmd2, nil
}

//func getNameByPid(pid uint32) (string, error) {
//	handle, err := syscall.CreateToolhelp32Snapshot(
//		syscall.TH32CS_SNAPMODULE,
//		pid,
//	)
//	if err != nil {
//		return "", fmt.Errorf("failed to create snapshot: %v", err)
//	}
//	defer syscall.CloseHandle(handle)
//
//	var me win.ModuleEntry32
//	me.Size = uint32(unsafe.Sizeof(me))
//	err = win.Module32First(win.Handle(handle), &me)
//	if err != nil {
//		return win.UTF16PtrToString(&me.Module[0]), nil
//	} else {
//		return "", fmt.Errorf("failed to get process entry: %v", err)
//	}
//}

func getNameByPid(pid uint32) (string, error) {

	handle, err := syscall.CreateToolhelp32Snapshot(
		syscall.TH32CS_SNAPPROCESS,
		// This argument will be ignored with a SNAPPROCESS flag.
		// The snapshot will contain all processes, we will need
		// to iterate over them and filter the desired one.
		0,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create snapshot: %v", err)
	}
	defer syscall.CloseHandle(handle)


	var cmd2 string

	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	err = syscall.Process32First(handle, &pe)
	if err != nil {
		return "", fmt.Errorf("failed to get process entry: %v", err)
	}

	if pe.ProcessID == pid {
		cmd2  = syscall.UTF16ToString(pe.ExeFile[:])
		return  cmd2,nil
	} else {
		for {
			var pe syscall.ProcessEntry32
			pe.Size = uint32(unsafe.Sizeof(pe))
			err = syscall.Process32Next(handle, &pe)
			if err != nil {
				return "", fmt.Errorf("failed to get next process entry: %v", err)
			}
			//	fmt.Println(pe.ProcessID, "  ",syscall.UTF16ToString(pe.ExeFile[:]))
			if  pe.ProcessID == pid {
				//ppid = int(pe.ParentProcessID)
				cmd2  = syscall.UTF16ToString(pe.ExeFile[:])
			//	fmt.Println("now  ppid ",ppid, " cmd2 ", cmd2)
				return cmd2,nil
			}
		}
	}
}

func getTcp4Table() (*win.MIB_TCPTABLE_OWNER_PID, error) {
	var size uint32 = 2 * 1024
	table := make([]byte, size)
	for {
		ret := win.GetExtendedTcpTable(
			uintptr(unsafe.Pointer(&table[0])),
			&size,
			int32(0),
			win.AF_INET,
			win.TCP_TABLE_OWNER_PID_ALL,
		)
		if ret == 0 {
			break
		} else if ret == win.ERROR_INSUFFICIENT_BUFFER {
			table = make([]byte, size)
			continue
		} else {
			return nil, fmt.Errorf("ret: %d", int(ret))
		}
	}
	return (*win.MIB_TCPTABLE_OWNER_PID)(unsafe.Pointer(&table[0])), nil
}

func getUdp4Table() (*win.MIB_UDPTABLE_OWNER_PID, error) {
	var size uint32 = 2 * 1024
	table := make([]byte, size)
	for {
		ret := win.GetExtendedUdpTable(
			uintptr(unsafe.Pointer(&table[0])),
			&size,
			int32(0),
			win.AF_INET,
			win.UDP_TABLE_OWNER_PID,
		)
		if ret == 0 {
			break
		} else if ret == win.ERROR_INSUFFICIENT_BUFFER {
			table = make([]byte, size)
			continue
		} else if ret != 0 {
			return nil, fmt.Errorf("ret: %d", int(ret))
		}
	}
	return (*win.MIB_UDPTABLE_OWNER_PID)(unsafe.Pointer(&table[0])), nil
}
