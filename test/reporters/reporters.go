package reporters

import (
	"flag"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	// reportDir is used to set the output directory for JUnit artifacts
	reportDir string
)

func init() {
	flag.StringVar(&reportDir, "report-dir", "", "Set report directory for artifact output")
}

// Reporters creates the ginkgo reporters for the test suites
func Reporters() []ginkgo.Reporter {
	now, _ := time.Now().MarshalText()
	reps := []ginkgo.Reporter{envtest.NewlineReporter{}}
	if reportDir != "" {
		reps = append(reps, reporters.NewJUnitReporter(fmt.Sprintf("%s/junit_%s_%d.xml", reportDir, string(now), config.GinkgoConfig.ParallelNode)))
	}
	return reps
}
