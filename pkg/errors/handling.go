package errors

import (
	"github.com/sirupsen/logrus"
)

func CheckErrorAndPanic(e error) {
	if e != nil {
		logrus.WithError(e)
		panic(e)
	}
}

func CheckErrorAndLog(e error) {
	if e != nil {
		logrus.WithError(e)
	}
}
