package types

import (
	clientTypes "k8s.io/apimachinery/pkg/types"
)

// ExperimentDetails is for collecting all the experiment-related details
type ExperimentDetails struct {
	ExperimentName   string
	EngineName       string
	ChaosDuration    int
	ChaosInterval    int
	RampTime         int
	ChaosLib         string
	AppNS            string
	AppLabel         string
	AppKind          string
	ChaosUID         clientTypes.UID
	InstanceID       string
	ChaosNamespace   string
	ChaosPodName     string
	Timeout          int
	Delay            int
	TargetContainer  string
	HostName         string
	HostDatacenter   string
	VcenterServer    string
	VcenterUser      string
	VcenterPass      string
	AuxiliaryAppInfo string
}