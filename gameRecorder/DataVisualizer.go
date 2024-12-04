package gameRecorder

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	// "strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// Add these constants at the top of the file
const (
	deathSymbol    = "💀"
	showLegends    = false   // Toggle for showing/hiding legends
	showAxisLabels = true    // Keep axis labels visible
	chartWidth     = "800px" // Increased from 500px
	chartHeight    = "500px" // Increased from 400px
)

// CreatePlaybackHTML generates visualizations for the recorded game data
func CreatePlaybackHTML(recorder *ServerDataRecorder) {
	// Create output directory if it doesn't exist
	outputDir := "visualization_output"
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Printf("Error creating output directory: %v\n", err)
		return
	}

	// Create a single page to hold all charts
	page := components.NewPage()
	page.PageTitle = "Game Visualization"

	// Add custom CSS for layout
	page.PageTitle = `
		<title>Game Visualization</title>
		<style>
			.chart-container { 
				display: flex; 
				flex-wrap: wrap; 
				justify-content: space-between; 
				margin: 20px;
			}
			.chart { 
				width: 48%; 
				min-width: 800px;  // Match chartWidth
				margin-bottom: 40px;  // Increased spacing between charts
			}
		</style>
		<div class="chart-container">
	`

	// Group turn records by iteration
	iterationMap := make(map[int][]TurnRecord)
	for _, record := range recorder.TurnRecords {
		iterationMap[record.IterationNumber] = append(iterationMap[record.IterationNumber], record)
	}

	// For each iteration, create side-by-side charts
	for iteration, turns := range iterationMap {
		scoreChart := createScoreChart(iteration, turns)
		contributionChart := createContributionChart(iteration, turns)

		// Add both charts to the page
		page.AddCharts(scoreChart, contributionChart)
	}

	// Create score plots
	createScorePlots(recorder, outputDir)

	// Create contribution plots
	createContributionPlots(recorder, outputDir)

	// Create the output file
	filepath := filepath.Join(outputDir, "game_visualization.html")
	f, err := os.Create(filepath)
	if err != nil {
		log.Printf("Error creating visualization file: %v\n", err)
		return
	}
	defer f.Close()

	// Render the page
	err = page.Render(f)
	if err != nil {
		panic(err)
	}
}

func createScoreChart(iteration int, turns []TurnRecord) *charts.Line {
	// Sort turns by turn number to ensure correct order
	sort.Slice(turns, func(i, j int) bool {
		return turns[i].TurnNumber < turns[j].TurnNumber
	})

	// Find first turn with agent records
	var initialAgentRecords []AgentRecord
	for _, turn := range turns {
		if len(turn.AgentRecords) > 0 {
			initialAgentRecords = turn.AgentRecords
			break
		}
	}

	if len(initialAgentRecords) == 0 {
		log.Printf("Warning: No agent records found in iteration %d\n", iteration)
		return nil
	}

	// Create a new line chart with adjusted layout
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: fmt.Sprintf("Iteration %d - Agent Scores over Time", iteration),
			Top:   "5%",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(showLegends),
			Top:  "15%",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name:    "Turn Number",
			NameGap: 30,
			AxisLabel: &opts.AxisLabel{
				Show: opts.Bool(showAxisLabels),
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name:    "Score",
			NameGap: 30,
			AxisLabel: &opts.AxisLabel{
				Show: opts.Bool(showAxisLabels),
			},
		}),
		charts.WithGridOpts(opts.Grid{
			Top:          "25%",
			Right:        "5%",
			Left:         "10%",
			Bottom:       "15%",
			ContainLabel: opts.Bool(true),
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  chartWidth,
			Height: chartHeight,
		}),
	)

	// Get turn numbers for X-axis
	xAxis := make([]int, len(turns))
	for i, turn := range turns {
		xAxis[i] = turn.TurnNumber
	}

	// Create a map of agent scores over turns
	agentScores := make(map[string][]float64)
	for _, agent := range initialAgentRecords {
		agentID := agent.AgentID.String()
		agentScores[agentID] = make([]float64, len(turns))
	}

	// Fill in the scores
	for turnIdx, turn := range turns {
		for _, agent := range turn.AgentRecords {
			agentID := agent.AgentID.String()
			agentScores[agentID][turnIdx] = float64(agent.Score)
		}
	}

	// Create a map to store team colors
	teamColors := make(map[string]string)
	for _, agent := range initialAgentRecords {
		agentID := agent.AgentID.String()
		teamColors[agentID] = getTeamColor(agent.TrueSomasTeamID)
	}

	// Add series and death markers
	for agentID, scores := range agentScores {
		var deathMarker opts.ScatterData
		var deathTurn int = -1

		// Find death turn
		for i, turn := range turns {
			for _, agent := range turn.AgentRecords {
				if agent.AgentID.String() == agentID && !agent.IsAlive {
					deathTurn = i
					deathMarker = opts.ScatterData{
						Value:      []interface{}{xAxis[i], scores[i]},
						Symbol:     "💀",
						SymbolSize: 20,
					}
					break
				}
			}
			if deathTurn != -1 {
				break
			}
		}

		// Truncate scores after death
		if deathTurn != -1 {
			scores = scores[:deathTurn+1]
		}

		// Add the series
		line.AddSeries(agentID, generateLineItems(xAxis[:len(scores)], scores),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: teamColors[agentID],
			}),
		)

		// Add death marker
		if deathTurn != -1 {
			scatter := charts.NewScatter()
			scatter.AddSeries(agentID+" Death", []opts.ScatterData{deathMarker})
			line.Overlap(scatter)
		}
	}

	line.SetXAxis(xAxis)
	return line
}

func createContributionChart(iteration int, turns []TurnRecord) *charts.Line {
	// Sort turns by turn number
	sort.Slice(turns, func(i, j int) bool {
		return turns[i].TurnNumber < turns[j].TurnNumber
	})

	// Find first turn with agent records
	var initialAgentRecords []AgentRecord
	for _, turn := range turns {
		if len(turn.AgentRecords) > 0 {
			initialAgentRecords = turn.AgentRecords
			break
		}
	}

	if len(initialAgentRecords) == 0 {
		log.Printf("Warning: No agent records found in iteration %d\n", iteration)
		return nil
	}

	// Create a new line chart
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: fmt.Sprintf("Iteration %d - Agent Contributions over Time", iteration),
			Top:   "5%",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   opts.Bool(true), // Enable the legend
			Top:    "15%",           // Adjust the position
			Left:   "right",         // Place legend on the right side
			Orient: "vertical",      // Display legend vertically
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name:    "Turn Number",
			NameGap: 30,
			AxisLabel: &opts.AxisLabel{
				Show: opts.Bool(true),
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name:    "Contribution",
			NameGap: 30,
			AxisLabel: &opts.AxisLabel{
				Show: opts.Bool(true),
			},
		}),
		charts.WithGridOpts(opts.Grid{
			Top:          "25%",
			Right:        "15%", // Add more space for the legend
			Left:         "10%",
			Bottom:       "15%",
			ContainLabel: opts.Bool(true),
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  chartWidth,
			Height: chartHeight,
		}),
	)

	// Get turn numbers for X-axis
	xAxis := make([]int, len(turns))
	for i, turn := range turns {
		xAxis[i] = turn.TurnNumber
	}

	// Create a map to store team colors
	teamColors := make(map[string]string)
	for _, agent := range initialAgentRecords {
		agentID := agent.AgentID.String()
		// Extract the UUID part (assuming it is always the first part)
		// Use regular expression to find a valid UUID pattern
		if uuid := extractAgentType(agentID); uuid != "" {
			teamColors[uuid] = getTeamColor(agent.TrueSomasTeamID)
		}
	}

	agentIDsInLegend := make(map[string]bool)
	for _, agent := range initialAgentRecords {
		agentType := agent.AgentType
		if _, exists := agentIDsInLegend[agentType]; !exists {
			agentIDsInLegend[agentType] = true
		}
	}

	// For each agent, create contribution lines
	for _, initialAgent := range initialAgentRecords {
		agentID := initialAgent.AgentID.String()
		// Extract only the UUID from the agentID
		if uuid := extractAgentType(agentID); uuid != "" {
			actualNet := make([]float64, len(turns))
			statedNet := make([]float64, len(turns))
			var deathMarker opts.ScatterData
			var deathTurn int = -1

			for i, turn := range turns {
				for _, agent := range turn.AgentRecords {
					if agent.AgentID.String() == agentID {
						actualNet[i] = float64(agent.Contribution - agent.Withdrawal)
						statedNet[i] = float64(agent.StatedContribution - agent.StatedWithdrawal)

						if !agent.IsAlive {
							deathTurn = i
							deathMarker = opts.ScatterData{
								Value:      []interface{}{xAxis[i], actualNet[i]},
								Symbol:     "💀",
								SymbolSize: 20,
							}
							break
						}
					}
				}
				if deathTurn != -1 {
					break
				}
			}

			// Truncate data after death
			if deathTurn != -1 {
				actualNet = actualNet[:deathTurn+1]
				statedNet = statedNet[:deathTurn+1]
			}

			// Add actual contribution line if not already added
			if _, exists := agentIDsInLegend[uuid]; !exists {
				line.AddSeries(uuid, generateLineItems(xAxis[:len(actualNet)], actualNet),
					charts.WithLineStyleOpts(opts.LineStyle{
						Color: teamColors[uuid],
					}),
				)
			}

			// Add stated contribution line if not already added
			if _, exists := agentIDsInLegend[uuid]; !exists {
				line.AddSeries(uuid+" (Stated)", generateLineItems(xAxis[:len(statedNet)], statedNet),
					charts.WithLineStyleOpts(opts.LineStyle{
						Color: teamColors[uuid],
						Type:  "dashed",
					}),
				)
			}

			// Add death marker
			if deathTurn != -1 {
				scatter := charts.NewScatter()
				scatter.AddSeries(uuid+" Death", []opts.ScatterData{deathMarker},
					charts.WithItemStyleOpts(opts.ItemStyle{
						Color: "black",
					}),
				)
				line.Overlap(scatter)
			}
		}
	}

	line.SetXAxis(xAxis)
	return line
}

// Helper function to extract the agent type string from a given input
func extractAgentType(input string) string {
	// Define a list of known agent types
	agentTypes := []string{
		"MI_256_v1",
		"Base Agent",
		"Team2Agent",
		// Add more agent types here if necessary
	}

	// Iterate through the known agent types to find a match in the input
	for _, agentType := range agentTypes {
		// if strings.Contains(input, agentType) {
		// 	return agentType
		// }
		return agentType
	}
	return agentTypes[2]
}

// Helper function to get team-based colors
func getTeamColor(teamID int) string {
	// Define a color palette for teams
	teamColors := []string{
		"#1f77b4", // blue
		"#ff7f0e", // orange
		"#2ca02c", // green
		"#d62728", // red
		"#9467bd", // purple
		"#8c564b", // brown
		"#e377c2", // pink
		"#7f7f7f", // gray
	}
	return teamColors[teamID%len(teamColors)]
}

// createScorePlots creates line charts showing agent scores over time for each iteration
func createScorePlots(recorder *ServerDataRecorder, outputDir string) {
	// Add safety check at the start
	if len(recorder.TurnRecords) == 0 {
		log.Println("Warning: No turn records to visualize")
		return
	}

	// Group turn records by iteration
	iterationMap := make(map[int][]TurnRecord)
	for _, record := range recorder.TurnRecords {
		iterationMap[record.IterationNumber] = append(iterationMap[record.IterationNumber], record)
	}

	// Create a page to hold all iteration charts
	page := components.NewPage()
	page.PageTitle = "Agent Scores Per Iteration"

	// For each iteration, create a line chart
	for iteration, turns := range iterationMap {
		// Sort turns by turn number to ensure correct order
		sort.Slice(turns, func(i, j int) bool {
			return turns[i].TurnNumber < turns[j].TurnNumber
		})

		// Find first turn with agent records to initialize our agent map
		var initialAgentRecords []AgentRecord
		for _, turn := range turns {
			if len(turn.AgentRecords) > 0 {
				initialAgentRecords = turn.AgentRecords
				break
			}
		}

		if len(initialAgentRecords) == 0 {
			log.Printf("Warning: No agent records found in iteration %d\n", iteration)
			continue
		}

		// Create a new line chart with adjusted layout
		line := charts.NewLine()
		line.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{
				Title: fmt.Sprintf("Iteration %d - Agent Scores over Time", iteration),
				Top:   "5%", // Move title to top with some padding
			}),
			charts.WithTooltipOpts(opts.Tooltip{
				Show: opts.Bool(true),
			}),
			charts.WithLegendOpts(opts.Legend{
				Show: opts.Bool(false),
			}),
			charts.WithXAxisOpts(opts.XAxis{
				Name:    "Turn Number",
				NameGap: 30, // Add gap between axis and name
				AxisLabel: &opts.AxisLabel{
					Show:   opts.Bool(true),
					Margin: 20, // Add margin to labels
				},
			}),
			charts.WithYAxisOpts(opts.YAxis{
				Name:    "Score",
				NameGap: 30, // Add gap between axis and name
			}),
			charts.WithGridOpts(opts.Grid{
				Top:    "28%",
				Right:  "5%",
				Left:   "10%", // Add more space for Y-axis labels
				Bottom: "15%", // Add more space for X-axis labels
			}),
		)

		// Get turn numbers for X-axis
		xAxis := make([]int, len(turns))
		for i, turn := range turns {
			xAxis[i] = turn.TurnNumber
		}

		// Create a map of agent scores over turns
		agentScores := make(map[string][]float64)

		// Initialize the map with empty slices using the first found agents
		for _, agent := range initialAgentRecords {
			agentID := agent.AgentID.String()
			agentScores[agentID] = make([]float64, len(turns))
		}

		// Fill in the scores
		for turnIdx, turn := range turns {
			for _, agent := range turn.AgentRecords {
				agentID := agent.AgentID.String()
				agentScores[agentID][turnIdx] = float64(agent.Score)
			}
		}

		// Create a map to store team colors
		teamColors := make(map[string]string)
		for _, agent := range initialAgentRecords {
			agentID := agent.AgentID.String()
			teamColors[agentID] = getTeamColor(agent.TrueSomasTeamID)
		}

		// When adding series, include team-based colors and mark dead agents
		for agentID, scores := range agentScores {
			// Find when the agent died (if they did)
			var deathMarker opts.ScatterData
			var deathTurn int = -1

			// Find the turn where agent died
			for i, turn := range turns {
				for _, agent := range turn.AgentRecords {
					if agent.AgentID.String() == agentID {
						if !agent.IsAlive {
							deathTurn = i
							deathMarker = opts.ScatterData{
								Value:      []interface{}{xAxis[i], scores[i]},
								Symbol:     deathSymbol,
								SymbolSize: 20,
							}
							break
						}
					}
				}
				if deathTurn != -1 {
					break
				}
			}

			// Truncate scores after death
			if deathTurn != -1 {
				scores = scores[:deathTurn+1]
			}

			// Add the series with custom styling
			line.AddSeries(agentID, generateLineItems(xAxis[:len(scores)], scores),
				charts.WithLineStyleOpts(opts.LineStyle{
					Color: teamColors[agentID],
				}),
				charts.WithItemStyleOpts(opts.ItemStyle{
					Color: teamColors[agentID],
				}),
			)

			// Add death marker as a scatter plot overlay
			if deathTurn != -1 {
				scatter := charts.NewScatter()
				scatter.AddSeries(agentID+" Death", []opts.ScatterData{deathMarker},
					charts.WithItemStyleOpts(opts.ItemStyle{
						Color: "black",
					}),
				)
				line.Overlap(scatter)
			}
		}

		// Set X-axis data
		line.SetXAxis(xAxis)

		// Add the chart to the page
		page.AddCharts(line)

		// Update file creation
		filepath := filepath.Join(outputDir, "agent_scores.html")
		f, err := os.Create(filepath)
		if err != nil {
			log.Printf("Error creating score plots file: %v\n", err)
			return
		}
		defer f.Close()

		// Render the page
		err = page.Render(f)
		if err != nil {
			panic(err)
		}
	}
}

// New function to create contribution visualization
func createContributionPlots(recorder *ServerDataRecorder, outputDir string) {
	if len(recorder.TurnRecords) == 0 {
		log.Println("Warning: No turn records to visualize")
		return
	}

	page := components.NewPage()
	page.PageTitle = "Agent Contributions Per Iteration"

	// Group by iteration
	iterationMap := make(map[int][]TurnRecord)
	for _, record := range recorder.TurnRecords {
		iterationMap[record.IterationNumber] = append(iterationMap[record.IterationNumber], record)
	}

	for iteration, turns := range iterationMap {
		// Sort turns by turn number
		sort.Slice(turns, func(i, j int) bool {
			return turns[i].TurnNumber < turns[j].TurnNumber
		})

		// Find first turn with agent records
		var initialAgentRecords []AgentRecord
		for _, turn := range turns {
			if len(turn.AgentRecords) > 0 {
				initialAgentRecords = turn.AgentRecords
				break
			}
		}

		if len(initialAgentRecords) == 0 {
			log.Printf("Warning: No agent records found in iteration %d\n", iteration)
			continue
		}

		// Create a new line chart
		line := charts.NewLine()
		line.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{
				Title: fmt.Sprintf("Iteration %d - Agent Contributions over Time", iteration),
				Top:   "5%",
			}),
			charts.WithTooltipOpts(opts.Tooltip{
				Show: opts.Bool(true),
			}),
			charts.WithLegendOpts(opts.Legend{
				Show: opts.Bool(false),
			}),
			charts.WithXAxisOpts(opts.XAxis{
				Name:    "Turn Number",
				NameGap: 30,
				AxisLabel: &opts.AxisLabel{
					Show:   opts.Bool(true),
					Margin: 20,
				},
			}),
			charts.WithYAxisOpts(opts.YAxis{
				Name:    "Contribution",
				NameGap: 30,
			}),
			charts.WithGridOpts(opts.Grid{
				Top:    "35%",
				Right:  "5%",
				Left:   "10%",
				Bottom: "15%",
			}),
		)

		// Get turn numbers for X-axis
		xAxis := make([]int, len(turns))
		for i, turn := range turns {
			xAxis[i] = turn.TurnNumber
		}

		// Create a map to store team colors
		teamColors := make(map[string]string)
		for _, agent := range initialAgentRecords {
			agentID := agent.AgentID.String()
			teamColors[agentID] = getTeamColor(agent.TrueSomasTeamID)
		}

		// For each agent, create contribution lines
		for _, initialAgent := range initialAgentRecords {
			agentID := initialAgent.AgentID.String()
			actualNet := make([]float64, len(turns))
			statedNet := make([]float64, len(turns))
			var deathMarker opts.ScatterData
			var deathTurn int = -1

			for i, turn := range turns {
				for _, agent := range turn.AgentRecords {
					if agent.AgentID == initialAgent.AgentID {
						actualNet[i] = float64(agent.Contribution - agent.Withdrawal)
						statedNet[i] = float64(agent.StatedContribution - agent.StatedWithdrawal)

						if !agent.IsAlive {
							deathTurn = i
							deathMarker = opts.ScatterData{
								Value:      []interface{}{xAxis[i], actualNet[i]},
								Symbol:     deathSymbol,
								SymbolSize: 20,
							}
							break
						}
					}
				}
				if deathTurn != -1 {
					break
				}
			}

			// Truncate data after death
			if deathTurn != -1 {
				actualNet = actualNet[:deathTurn+1]
				statedNet = statedNet[:deathTurn+1]
			}

			// Add actual contribution line
			line.AddSeries(agentID+" (Actual)", generateLineItems(xAxis[:len(actualNet)], actualNet),
				charts.WithLineStyleOpts(opts.LineStyle{
					Color: teamColors[agentID],
				}),
			)

			// Add stated contribution line (dotted)
			line.AddSeries(agentID+" (Stated)", generateLineItems(xAxis[:len(statedNet)], statedNet),
				charts.WithLineStyleOpts(opts.LineStyle{
					Color: teamColors[agentID],
					Type:  "dashed",
				}),
			)

			// Add death marker
			if deathTurn != -1 {
				scatter := charts.NewScatter()
				scatter.AddSeries(agentID+" Death", []opts.ScatterData{deathMarker},
					charts.WithItemStyleOpts(opts.ItemStyle{
						Color: "black",
					}),
				)
				line.Overlap(scatter)
			}
		}

		// Set X-axis data
		line.SetXAxis(xAxis)

		// Add the chart to the page
		page.AddCharts(line)

		// Update file creation
		filepath := filepath.Join(outputDir, "agent_contributions.html")
		f, err := os.Create(filepath)
		if err != nil {
			log.Printf("Error creating contribution plots file: %v\n", err)
			return
		}
		defer f.Close()

		err = page.Render(f)
		if err != nil {
			panic(err)
		}
	}
}

// Helper function to generate line chart items
func generateLineItems(xAxis []int, yAxis []float64) []opts.LineData {
	items := make([]opts.LineData, len(xAxis))
	for i := 0; i < len(xAxis); i++ {
		items[i] = opts.LineData{Value: yAxis[i]}
	}
	return items
}