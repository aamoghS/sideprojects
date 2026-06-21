package plans

import "fmt"

type Plan struct {
	ID     string `json:"id"`
	CPU    int    `json:"cpu"`
	Memory int    `json:"memory_mb"`
	Disk   int    `json:"disk_gb"`
}

var Default = map[string]Plan{
	"tiny":   {ID: "tiny", CPU: 1, Memory: 512, Disk: 10},
	"small":  {ID: "small", CPU: 1, Memory: 1024, Disk: 20},
	"medium": {ID: "medium", CPU: 2, Memory: 2048, Disk: 40},
	"large":  {ID: "large", CPU: 4, Memory: 4096, Disk: 80},
}

func Get(id string) (Plan, error) {
	if p, ok := Default[id]; ok {
		return p, nil
	}
	return Plan{}, fmt.Errorf("unknown plan %q (try tiny, small, medium, large)", id)
}

func List() []Plan {
	out := make([]Plan, 0, len(Default))
	for _, p := range Default {
		out = append(out, p)
	}
	return out
}
