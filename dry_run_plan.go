package main

type DryRunPlan struct {
	Version  int                `json:"version"`
	Mode     string             `json:"mode"`
	Job      DryRunPlanJob      `json:"job"`
	Targets  []string           `json:"targets"`
	Warnings []string           `json:"warnings"`
	Workers  []DryRunPlanWorker `json:"workers"`
}

type DryRunPlanJob struct {
	Name      string `json:"name"`
	Framework string `json:"framework"`
	Reason    string `json:"reason"`
}

type DryRunPlanWorker struct {
	Index   int      `json:"index"`
	Targets []string `json:"targets"`
	Argv    []string `json:"argv"`
	Env     []string `json:"env"`
	Shell   string   `json:"shell"`
}

type RunnerDryRunPlan struct {
	Targets []string
	Workers []DryRunPlanWorker
}
