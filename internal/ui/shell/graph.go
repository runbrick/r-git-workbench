package shell

import (
	"strings"

	"github.com/vivaui/viva/ui"

	"r-git-workbench/internal/git"
)

type graphRow struct {
	lane   int
	before []string
	after  []string
	merge  []bool
}

// buildGraph keeps pending parents in lanes, so a page can extend the graph
// without restarting its tracks at the page boundary.
func buildGraph(commits []git.Commit) []graphRow {
	rows := make([]graphRow, len(commits))
	var lanes []string
	ensureLane := func(index int) {
		for len(lanes) <= index {
			lanes = append(lanes, "")
		}
	}
	for i, commit := range commits {
		lane := -1
		for j, sha := range lanes {
			if sha == commit.SHA {
				lane = j
				break
			}
		}
		if lane < 0 {
			for j, sha := range lanes {
				if sha == "" {
					lane = j
					break
				}
			}
		}
		if lane < 0 {
			lane = len(lanes)
		}
		ensureLane(lane)
		lanes[lane] = commit.SHA
		row := &rows[i]
		row.lane = lane
		row.before = append([]string(nil), lanes...)
		lanes[lane] = ""
		for p, parent := range commit.Parents {
			if parent == "" {
				continue
			}
			existing := -1
			for j, sha := range lanes {
				if sha == parent {
					existing = j
					break
				}
			}
			if existing >= 0 {
				if p > 0 {
					for len(row.merge) <= existing {
						row.merge = append(row.merge, false)
					}
					row.merge[existing] = true
				}
				continue
			}
			place := lane
			if p > 0 {
				place = -1
				for j := lane + 1; j < len(lanes); j++ {
					if lanes[j] == "" {
						place = j
						break
					}
				}
				if place < 0 {
					place = len(lanes)
				}
			}
			if place >= len(lanes) {
				ensureLane(place)
			}
			if place >= 0 && lanes[place] != "" {
				place = -1
				for j, sha := range lanes {
					if sha == "" {
						place = j
						break
					}
				}
			}
			if place >= 0 {
				ensureLane(place)
				lanes[place] = parent
			}
			if place >= 0 && p > 0 {
				for len(row.merge) <= place {
					row.merge = append(row.merge, false)
				}
				row.merge[place] = true
			}
		}
		row.after = append([]string(nil), lanes...)
	}
	return rows
}

var graphColors = [...]ui.Color{
	ui.RGB(184, 77, 120), ui.RGB(201, 151, 67), ui.RGB(99, 166, 178),
	ui.RGB(140, 115, 187), ui.RGB(92, 164, 116), ui.RGB(197, 115, 82),
}

func paintGraph(canvas *ui.Canvas, bounds ui.Rect, scroll float32, rows []graphRow) {
	const rowHeight float32 = 25
	first := max(int(scroll/rowHeight), 0)
	last := min(int((scroll+bounds.Height)/rowHeight)+2, len(rows))
	if first >= last || bounds.Height <= 0 {
		return
	}
	canvas.Clip(bounds)
	for i := first; i < last; i++ {
		row := rows[i]
		y := bounds.Y + float32(i)*rowHeight - scroll
		for lane, sha := range row.before {
			if sha != "" && lane != row.lane {
				graphLine(canvas, bounds.X+12+float32(lane)*12, y, bounds.X+12+float32(lane)*12, y+rowHeight, graphColor(lane))
			}
		}
		for lane, sha := range row.after {
			if sha == "" {
				continue
			}
			x := bounds.X + 12 + float32(lane)*12
			if lane == row.lane {
				graphLine(canvas, x, y+rowHeight/2, x, y+rowHeight, graphColor(lane))
			} else if lane < len(row.merge) && row.merge[lane] {
				graphLine(canvas, bounds.X+12+float32(row.lane)*12, y+rowHeight/2, x, y+rowHeight, graphColor(lane))
			}
		}
		x := bounds.X + 12 + float32(row.lane)*12
		if i > 0 && row.before[row.lane] != "" {
			graphLine(canvas, x, y, x, y+rowHeight/2, graphColor(row.lane))
		}
		canvas.Rounded(ui.Rect{X: x - 4, Y: y + rowHeight/2 - 4, Width: 8, Height: 8}, 4, graphColor(row.lane))
	}
	canvas.Unclip()
}

func graphColor(lane int) ui.Color { return graphColors[lane%len(graphColors)] }

func refLabels(sha string, refs []git.Ref, head string) string {
	labels := make([]string, 0, 4)
	if sha == head {
		labels = append(labels, "HEAD")
	}
	for _, ref := range refs {
		if ref.SHA != sha || len(labels) == cap(labels) {
			continue
		}
		prefix := ""
		if ref.Tag {
			prefix = "tag: "
		} else if ref.Remote {
			prefix = "remote: "
		} else {
			prefix = "branch: "
		}
		labels = append(labels, prefix+ref.Name)
	}
	if len(labels) == 0 {
		return ""
	}
	return "  [" + strings.Join(labels, ", ") + "]"
}

func graphLine(canvas *ui.Canvas, x1, y1, x2, y2 float32, color ui.Color) {
	builder := canvas.PathBuilder()
	x, y := min(x1, x2), min(y1, y2)
	w, h := max(x1, x2)-x+0.01, max(y1, y2)-y+0.01
	builder.MoveTo(x1-x, y1-y)
	builder.LineTo(x2-x, y2-y)
	canvas.DrawPath(builder.Path(w, h), ui.Rect{X: x, Y: y, Width: w, Height: h}, ui.PathStyle{Stroke: color, StrokeWidth: 1.5, Cap: ui.CapRound})
}
