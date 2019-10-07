// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package query

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

const (
	DateTimeFormat = "2006-01-02 15:04:05"
	DateFormat     = "2006-01-02"
)

type Validator func(*Atom) error

func NullValidator(*Atom) error {
	return nil
}

func RegexpValidator(re string) Validator {
	rex := regexp.MustCompile(re)

	return func(a *Atom) error {
		if !rex.MatchString(a.Value) {
			return fmt.Errorf("invalid value for %s: %s", a.Key, a.Value)
		}
		return nil
	}
}

func SetValidator(set ...string) Validator {
	s := make(map[string]struct{})
	for _, v := range set {
		s[v] = struct{}{}
	}

	return func(a *Atom) error {
		if _, ok := s[a.Value]; ok {
			return nil
		}

		// TODO add valid values
		return fmt.Errorf("invalid value for %s: %s", a.Key, a.Value)
	}
}

func DateValidator(a *Atom) error {
	if tm, err := time.Parse(time.RFC3339, a.Value); err == nil {
		a.Value = tm.UTC().Format(DateTimeFormat)
		return nil
	}

	if tm, err := time.Parse(DateTimeFormat, a.Value); err == nil {
		a.Value = tm.Format(DateTimeFormat)
		return nil
	}

	if tm, err := time.Parse(DateFormat, a.Value); err == nil {
		a.Value = tm.Format(DateFormat)
		return nil
	}

	return fmt.Errorf("invalid value for %s: %q", a.Key, a.Value)
}

func DomainValidator(a *Atom) error {
	u, err := idna.ToUnicode(a.Value)
	if err != nil {
		return fmt.Errorf("invalid value for %s: %s: %s", a.Key, a.Value, err)
	}
	a.Value = u

	a.Value = strings.TrimLeft(a.Value, ".")
	a.Value = strings.TrimRight(a.Value, ".")
	a.Value = regexp.MustCompile(`\.+`).ReplaceAllString(a.Value, ".")
	a.Value = strings.ToLower(a.Value)

	return nil
}

func URLValidator(a *Atom) error {
	u, err := url.Parse(a.Value)
	if err != nil {
		return fmt.Errorf("invalid value for %s: %s: %s", a.Key, a.Value, err)
	}
	a.Value = u.String()
	return nil
}

func IPValidator(a *Atom) error {
	switch a.Comparator {
	case CmpEqual, CmpNotEqual:
	default:
		return fmt.Errorf("invalid comparator for %s: %s", a.Key, a.Comparator)
	}

	_, cidr, err := net.ParseCIDR(a.Value)
	if err == nil {
		a.Value = cidr.String()
		return nil
	}

	ip := net.ParseIP(a.Value)
	if ip == nil {
		return fmt.Errorf("invalid value for %s: %s", a.Key, a.Value)
	}
	a.Value = ip.String()
	return nil
}

func IntRangeValidator(low, high int64) Validator {
	return func(a *Atom) error {
		i, err := strconv.ParseInt(a.Value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid value for %s: %s: %s", a.Key, a.Value, err)
		}

		if i < low || i > high {
			return fmt.Errorf("invalid value for %s: %s", a.Key, a.Value)
		}

		return nil
	}
}

func PositiveIntValidator(a *Atom) error {
	i, err := strconv.ParseInt(a.Value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid value for %s: %s", a.Key, a.Value)
	}

	if i < 0 {
		return fmt.Errorf("invalid value for %s: %d < 0", a.Key, i)
	}

	return nil
}
