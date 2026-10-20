package main

import "syscall"

// unameSRM is what uname -srm prints: kernel name, release and machine.
func unameSRM() string {
	var u syscall.Utsname
	if syscall.Uname(&u) != nil {
		return ""
	}
	return cstring(u.Sysname[:]) + " " + cstring(u.Release[:]) + " " + cstring(u.Machine[:])
}

func cstring[T int8 | uint8](a []T) string {
	b := make([]byte, 0, len(a))
	for _, c := range a {
		if c == 0 {
			break
		}
		b = append(b, byte(c))
	}
	return string(b)
}
