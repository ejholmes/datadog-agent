package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/soniah/gosnmp"
)

const (
	snmpCheckName = "snmp"
)

// Check aggregates metrics from one Check instance
type Check struct {
	core.CheckBase
	config  snmpConfig
	session snmpSession
}

// Run executes the check
func (c *Check) Run() error {
	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}

	sender.Gauge("snmp.test.metric", float64(10), "", nil)

	err = c.session.Connect()
	if err != nil {
		log.Errorf("Connect() err: %v", err)
	}
	defer c.session.Close() // TODO: handle error?

	oids := []string{"1.3.6.1.2.1.25.6.3.1.5.130"}
	result, err := c.session.Get(oids)
	if err != nil {
		log.Errorf("Get() err: %v", err)
		return nil
	}

	for j, variable := range result.Variables {
		log.Infof("%d: oid: %s ", j, variable.Name)
		switch variable.Type {
		case gosnmp.OctetString:
			log.Infof("string: %s\n", string(variable.Value.([]byte)))
		default:
			log.Infof("number: %d\n", gosnmp.ToBigInt(variable.Value))
		}
	}

	log.Debug("Run snmp")

	sender.Commit()

	return nil
}

// Configure configures the snmp checks
func (c *Check) Configure(rawInstance integration.Data, rawInitConfig integration.Data, source string) error {
	err := c.CommonConfigure(rawInstance, source)
	if err != nil {
		return err
	}
	config, err := buildConfig(rawInstance, rawInitConfig)
	if err != nil {
		return err
	}
	c.config = config
	c.session = buildSession(c.config)

	return nil
}

func snmpFactory() check.Check {
	return &Check{
		CheckBase: core.NewCheckBase(snmpCheckName),
	}
}

func init() {
	core.RegisterCheck(snmpCheckName, snmpFactory)
}
