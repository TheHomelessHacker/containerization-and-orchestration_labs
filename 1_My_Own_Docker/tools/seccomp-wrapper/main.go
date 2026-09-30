package main

import (
	"fmt"
	"os"
	"syscall"

	seccomp "github.com/seccomp/libseccomp-golang"
)

func main() {
	// 
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: myseccomp <command> [args...]")
		os.Exit(1)
	}

	if err := applySeccomp(); err != nil {
		fmt.Fprintln(os.Stderr, "seccomp apply failed:", err)
		os.Exit(1)
	}

	// Заменить себя на целевой процесс — фильтр наследуется
	if err := syscall.Exec(os.Args[1], os.Args[1:], os.Environ()); err != nil {
		fmt.Fprintln(os.Stderr, "exec failed:", err)
		os.Exit(1)
	}
}

func applySeccomp() error {
	// Создаём фильтр. По умолчанию — SCMP_ACT_ALLOW (разрешить всё)
	filter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		return fmt.Errorf("NewFilter: %w", err)
	}
	defer filter.Release()

	blocked := []string{
		"mount",         // монтирование ФС
		"umount2",       // размонтирование
		"ptrace",        // отладка чужих процессов
		"kexec_load",    // загрузка своего ядра
		"bpf",           // программирование eBPF
		"keyctl",        // работа с keyring
		"reboot",        // перезагрузка системы
		"init_module",   // загрузка модуля ядра
		"delete_module", // выгрузка модуля ядра
		"swapon",        // включение swap
		"swapoff",       // выключение swap
	}

	for _, name := range blocked {
		syscallID, err := seccomp.GetSyscallFromName(name)
		if err != nil {
			// Если syscall нет в данной архитектуре — пропускаем
			continue
		}
		if err := filter.AddRule(syscallID, seccomp.ActErrno.SetReturnCode(int16(syscall.EPERM))); err != nil {
			return fmt.Errorf("AddRule %s: %w", name, err)
		}
	}

	// Применяем фильтр к текущему процессу
	if err := filter.Load(); err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	return nil
}
