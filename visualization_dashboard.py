import pandas as pd
import dash
from dash import dcc, html
from dash.dependencies import Input, Output
import plotly.express as px
import plotly.graph_objects as go
from datetime import datetime
import hashlib
import json

# Add these constants at the top of the file for consistent plot styling
PLOT_HEIGHT = 600  # Increased height
PLOT_WIDTH = 1000  # Increased from 700 to 1000
PLOT_MARGIN = dict(r=150)  # More space on the right for legends

# Add these global variables after the PLOT constants
last_data_hash = None
cached_data = None


# Move the data loading into a function
def load_data():
    global last_data_hash, cached_data

    base_path = "visualization_output/csv_data"
    current_data = {
        "agent_records": pd.read_csv(f"{base_path}/agent_records.csv"),
        "common_records": pd.read_csv(f"{base_path}/common_records.csv"),
        "team_records": pd.read_csv(f"{base_path}/team_records.csv"),
        "team1_boundaries": pd.read_csv(f"{base_path}/team1Rank_records.csv"),
    }

    # Create a hash of the current data
    hash_string = ""
    for df in current_data.values():
        hash_string += df.to_json()
    current_hash = hashlib.md5(hash_string.encode()).hexdigest()

    # If the hash matches, return cached data
    if last_data_hash == current_hash and cached_data is not None:
        return cached_data

    # Otherwise, update cache and return new data
    last_data_hash = current_hash
    cached_data = current_data
    return current_data


# Load initial data
data = load_data()
agent_records = data["agent_records"]

# Initialize the Dash app
app = dash.Dash(__name__)

team_filter = dcc.Dropdown(
    id="team-filter",
    options=[
        {"label": f"Team {i}", "value": i}
        for i in data["agent_records"]["TrueSomasTeamID"].unique()
    ],
    value=1,  # Default to Team 1
    clearable=False
)

team_boundaries_filter = dcc.Dropdown(
    id="team-boundaries-filter",
    options=[
        {"label": tid[:8], "value": tid}
        for tid in data["team1_boundaries"]["TeamID"].unique()
        if tid != "00000000-0000-0000-0000-000000000000"
    ],
    value=data["team1_boundaries"]["TeamID"].unique()[1],  # Select a default non-empty team ID
    clearable=False
)

# Create the layout
app.layout = html.Div(
    [
        dcc.Interval(
            id="interval-component",
            interval=5000 * 1000,  # in milliseconds (5 seconds)
            n_intervals=0,
        ),
        html.H1("Agent Performance Dashboard"),
        html.Div(
            [
                html.H3("Filters"),
                dcc.Dropdown(
                    id="iteration-filter",
                    options=[
                        {"label": f"Iteration {i}", "value": i}
                        for i in agent_records["IterationNumber"].unique()
                    ],
                    value=0,
                    clearable=False,
                ),
            ],
            style={"width": "200px", "margin": "20px"},
        ),
        html.Div(
            [
                html.Div(
                    [
                        dcc.Graph(id="score-evolution"),
                    ],
                    className="graph-container",
                ),
                html.Div(
                    [
                        dcc.Graph(id="score-evolution-by-team"),
                    ],
                    className="graph-container",
                ),
                html.Div(
                    [
                        dcc.Graph(id="common-pool"),
                    ],
                    className="graph-container",
                ),
                html.Div(
                    [
                        dcc.Graph(id="agent-status"),
                    ],
                    className="graph-container",
                ),
                html.Div(
                    [
                        dcc.Graph(id="individual-net-contributions"),
                    ],
                    className="graph-container",
                ),
            ]
        ),
        # Add this to the layout section of your dashboard:
        html.Div([
            dcc.Graph(
                id="team-1",
                className="graph-container"
            )
        ], className="graph-section"),
        html.Div([
            html.H3("Agent Contributions"),
            team_filter,
            dcc.Graph(id="contribution-plot")
        ]),

                # Add this section for boundaries:
        html.Div([
            html.H3("Team Boundaries"),
            team_boundaries_filter,
            dcc.Graph(id="boundaries-plot"),
        ], className="graph-section"),
    ]
)


# Callback for score evolution with threshold overlay
@app.callback(
    Output("score-evolution", "figure"),
    [Input("iteration-filter", "value"), Input("interval-component", "n_intervals")],
)
def update_score_evolution(iteration, n_intervals):
    # Reload data each time
    data = load_data()
    filtered_data = data["agent_records"][
        data["agent_records"]["IterationNumber"] == iteration
    ]
    threshold_data = data["common_records"][
        data["common_records"]["IterationNumber"] == iteration
    ]

    fig = go.Figure()

    # Get all unique TrueSomasTeamIDs for creating buttons
    unique_teams = sorted(filtered_data["TrueSomasTeamID"].unique())

    # Group agents by their true team ID
    for team_id in filtered_data["TrueSomasTeamID"].unique():
        team_data = filtered_data[filtered_data["TrueSomasTeamID"] == team_id]

        # Create a consistent color for this team
        team_color = px.colors.qualitative.Set1[
            team_id % len(px.colors.qualitative.Set1)
        ]

        # Plot each agent in the team with the same color
        for agent_id in team_data["AgentID"].unique():
            agent_data = team_data[team_data["AgentID"] == agent_id]

            # Find where agent dies (if it does)
            death_point = (
                agent_data[agent_data["IsAlive"] == False].iloc[0]
                if any(~agent_data["IsAlive"])
                else None
            )

            # Plot including the death point
            if death_point is not None:
                # Include all data up to and including death point
                plot_data = agent_data[
                    agent_data["TurnNumber"] <= death_point["TurnNumber"]
                ]
            else:
                plot_data = agent_data

            if not plot_data.empty:
                # Get the first row of agent_data to extract consistent metadata
                agent_info = agent_data.iloc[0]
                fig.add_trace(
                    go.Scatter(
                        x=plot_data["TurnNumber"],
                        y=plot_data["Score"],
                        name=f'T{team_id}_{agent_info["TeamID"][:4]}_{agent_id[:8]}',
                        mode="lines",
                        line=dict(color=team_color),
                    )
                )

            # Add skull emoji at death point if agent dies
            if death_point is not None:
                fig.add_trace(
                    go.Scatter(
                        x=[death_point["TurnNumber"]],
                        y=[death_point["Score"]],
                        mode="text",
                        text=["💀"],
                        textfont=dict(size=20),
                        showlegend=False,
                        hoverinfo="skip",
                    )
                )

    # Add threshold line
    fig.add_trace(
        go.Scatter(
            x=threshold_data["TurnNumber"] + 1,  # Add 1 to shift right
            y=threshold_data["Threshold"],
            name="Threshold",
            mode="lines",
            line=dict(color="red", dash="dash"),
        )
    )

    # Add vertical lines for threshold application turns
    threshold_turns = threshold_data[threshold_data["ThresholdAppliedInTurn"] == True][
        "TurnNumber"
    ]
    for turn in threshold_turns:
        fig.add_vline(x=turn, line_dash="dash", line_color="red", opacity=0.5)

    # Add buttons for team visibility
    updatemenus = [
        dict(
            type="dropdown",
            direction="down",
            x=-0.1,
            y=1.2,
            xanchor="left",
            yanchor="top",
            showactive=True,
            bgcolor="rgb(238, 238, 238)",
            bordercolor="rgb(200, 200, 200)",
            borderwidth=1,
            pad={"r": 10, "t": 10},
            buttons=[
                # Button to show all teams
                dict(
                    label="All Teams",
                    method="update",
                    args=[{"visible": [True] * len(fig.data)}],
                ),
            ]
            +
            # Button for each team
            [
                dict(
                    label=f"Team {team_id}",
                    method="update",
                    args=[
                        {
                            "visible": [
                                # For each trace, handle special cases and team filtering
                                True
                                if not hasattr(trace, "name")
                                or trace.mode == "text"  # Keep death markers
                                else True
                                if getattr(trace, "name", None)
                                == "Threshold"  # Keep threshold line
                                else True
                                if getattr(trace, "name", None)
                                and trace.name.startswith(f"T{team_id}_")  # Check team
                                else False
                                for trace in fig.data
                            ]
                        }
                    ],
                )
                for team_id in unique_teams
            ],
            active=0,
            font={"size": 12},
        )
    ]

    # Update layout with buttons
    fig.update_layout(
        title=dict(
            text="Score Evolution Over Time - By True Team",
            y=0.95,
            x=0.5,
            xanchor="center",
            yanchor="top",
        ),
        margin=dict(t=150, r=150, b=50, l=50),
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        hovermode="x unified",
        showlegend=True,
        legend=dict(
            groupclick="toggleitem", yanchor="top", y=0.99, xanchor="left", x=1.05
        ),
        updatemenus=updatemenus,
    )

    return fig


# Callback for common pool evolution
@app.callback(Output("common-pool", "figure"), [Input("iteration-filter", "value")])
def update_common_pool(iteration):
    filtered_data = data["team_records"][
        data["team_records"]["IterationNumber"] == iteration
    ]

    fig = go.Figure()

    for team_id in filtered_data["TeamID"].unique():
        team_data = filtered_data[filtered_data["TeamID"] == team_id]
        if team_id != "00000000-0000-0000-0000-000000000000":  # Skip empty team
            fig.add_trace(
                go.Scatter(
                    x=team_data["TurnNumber"],
                    y=team_data["TeamCommonPool"],
                    name=f"Team {team_id[:8]}",
                    mode="lines+markers",
                )
            )

    fig.update_layout(
        title="Team Common Pool Evolution",
        xaxis_title="Turn Number",
        yaxis_title="Common Pool Value",
        hovermode="x unified",
        showlegend=True,
        legend=dict(yanchor="top", y=0.99, xanchor="left", x=1.05),
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        margin=dict(t=150, r=150, b=50, l=50),
    )

    return fig


# Callback for agent status
@app.callback(Output("agent-status", "figure"), [Input("iteration-filter", "value")])
def update_agent_status(iteration):
    filtered_data = agent_records[agent_records["IterationNumber"] == iteration]

    status_counts = (
        filtered_data.groupby(["TurnNumber", "IsAlive"]).size().unstack(fill_value=0)
    )

    fig = go.Figure()

    # Safely get alive counts (True), defaulting to 0 if not present
    alive_counts = (
        status_counts[True]
        if True in status_counts.columns
        else pd.Series(0, index=status_counts.index)
    )

    # Safely get dead counts (False), defaulting to 0 if not present
    dead_counts = (
        status_counts[False]
        if False in status_counts.columns
        else pd.Series(0, index=status_counts.index)
    )

    fig.add_trace(
        go.Bar(
            x=status_counts.index, y=alive_counts, name="Alive", marker_color="green"
        )
    )

    fig.add_trace(
        go.Bar(x=status_counts.index, y=dead_counts, name="Dead", marker_color="red")
    )

    fig.update_layout(
        title="Agent Status Over Time",
        xaxis_title="Turn Number",
        yaxis_title="Number of Agents",
        barmode="stack",
        showlegend=True,
        legend=dict(yanchor="top", y=0.99, xanchor="left", x=1.05),
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        margin=dict(t=150, r=150, b=50, l=50),
    )

    return fig


# Add new callback for team-based score evolution
@app.callback(
    Output("score-evolution-by-team", "figure"),
    [Input("iteration-filter", "value"), Input("interval-component", "n_intervals")],
)
def update_score_evolution_by_team(iteration, n_intervals):
    # Reload data each time
    data = load_data()
    filtered_data = data["agent_records"][
        data["agent_records"]["IterationNumber"] == iteration
    ]
    threshold_data = data["common_records"][
        data["common_records"]["IterationNumber"] == iteration
    ]

    fig = go.Figure()

    # Get all unique TeamIDs for creating buttons
    unique_teams = sorted(filtered_data["TeamID"].unique())

    # Group agents by their TeamID instead of TrueSomasTeamID
    for idx, (team_id, team_data) in enumerate(filtered_data.groupby("TeamID")):
        # Create a consistent color for this team
        team_color = px.colors.qualitative.Set1[idx % len(px.colors.qualitative.Set1)]

        # Plot each agent in the team
        for agent_id in team_data["AgentID"].unique():
            agent_data = team_data[team_data["AgentID"] == agent_id]

            # Find where agent dies (if it does)
            death_point = None
            if any(~agent_data["IsAlive"]):
                death_data = agent_data[~agent_data["IsAlive"]]
                if not death_data.empty:
                    death_point = death_data.iloc[0]

            # Plot including the death point
            if death_point is not None:
                # Include all data up to and including death point
                plot_data = agent_data[
                    agent_data["TurnNumber"] <= death_point["TurnNumber"]
                ]
            else:
                plot_data = agent_data

            if not plot_data.empty:
                # Get the first row of agent_data to extract consistent metadata
                agent_info = agent_data.iloc[0]
                fig.add_trace(
                    go.Scatter(
                        x=plot_data["TurnNumber"],
                        y=plot_data["Score"],
                        name=f'T{agent_info["TrueSomasTeamID"]}_{team_id[:4]}_{agent_id[:8]}',
                        mode="lines",
                        line=dict(color=team_color),
                    )
                )

            # Add skull emoji at death point if agent dies
            if death_point is not None:
                fig.add_trace(
                    go.Scatter(
                        x=[death_point["TurnNumber"]],
                        y=[death_point["Score"]],
                        mode="text",
                        text=["💀"],
                        textfont=dict(size=20),
                        showlegend=False,
                        hoverinfo="skip",
                    )
                )

    # Add threshold line
    fig.add_trace(
        go.Scatter(
            x=threshold_data["TurnNumber"] + 1,  # Add 1 to shift right
            y=threshold_data["Threshold"],
            name="Threshold",
            mode="lines",
            line=dict(color="red", dash="dash"),
        )
    )

    # Add vertical lines for threshold application turns
    threshold_turns = threshold_data[threshold_data["ThresholdAppliedInTurn"] == True][
        "TurnNumber"
    ]
    for turn in threshold_turns:
        fig.add_vline(x=turn, line_dash="dash", line_color="red", opacity=0.5)

    # Add buttons for team visibility
    updatemenus = [
        dict(
            type="dropdown",
            direction="down",
            x=-0.1,
            y=1.2,
            xanchor="left",
            yanchor="top",
            showactive=True,
            bgcolor="rgb(238, 238, 238)",
            bordercolor="rgb(200, 200, 200)",
            borderwidth=1,
            pad={"r": 10, "t": 10},
            buttons=[
                # Button to show all teams
                dict(
                    label="All Teams",
                    method="update",
                    args=[{"visible": [True] * len(fig.data)}],
                ),
            ]
            +
            # Button for each team
            [
                dict(
                    label=f"Team {team_id[:4]}",
                    method="update",
                    args=[
                        {
                            "visible": [
                                # For each trace, handle special cases and team filtering
                                True
                                if not hasattr(trace, "name")
                                or trace.mode == "text"  # Keep death markers
                                else True
                                if getattr(trace, "name", None)
                                == "Threshold"  # Keep threshold line
                                else True
                                if getattr(trace, "name", None)
                                and team_id[:4] in trace.name  # Check team
                                else False
                                for trace in fig.data
                            ]
                        }
                    ],
                )
                for team_id in unique_teams
            ],
            active=0,
            font={"size": 12},
        )
    ]

    # Update layout with buttons
    fig.update_layout(
        title=dict(
            text="Score Evolution Over Time - By Team",
            y=0.95,
            x=0.5,
            xanchor="center",
            yanchor="top",
        ),
        margin=dict(t=150, r=150, b=50, l=50),
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        hovermode="x unified",
        showlegend=True,
        legend=dict(
            groupclick="toggleitem", yanchor="top", y=0.99, xanchor="left", x=1.05
        ),
        updatemenus=updatemenus,
    )

    return fig


# Add new callback for individual net contributions
@app.callback(
    Output("individual-net-contributions", "figure"),
    [Input("iteration-filter", "value"), Input("interval-component", "n_intervals")],
)
def update_individual_net_contributions(iteration, n_intervals):
    # Reload data each time
    data = load_data()
    filtered_data = data["agent_records"][
        data["agent_records"]["IterationNumber"] == iteration
    ]

    fig = go.Figure()

    # Group agents by their true team ID
    for team_id in filtered_data["TrueSomasTeamID"].unique():
        team_data = filtered_data[filtered_data["TrueSomasTeamID"] == team_id]
        team_color = px.colors.qualitative.Set1[
            team_id % len(px.colors.qualitative.Set1)
        ]

        # Plot each agent in the team
        for agent_id in team_data["AgentID"].unique():
            agent_data = team_data[team_data["AgentID"] == agent_id]

            # Find where agent dies (if it does)
            death_point = None
            if any(~agent_data["IsAlive"]):
                death_data = agent_data[~agent_data["IsAlive"]]
                if not death_data.empty:
                    death_point = death_data.iloc[0]

            # Plot including the death point
            if death_point is not None:
                plot_data = agent_data[
                    agent_data["TurnNumber"] <= death_point["TurnNumber"]
                ]
            else:
                plot_data = agent_data

            if not plot_data.empty:
                agent_info = agent_data.iloc[0]
                # Calculate net contribution (contribution minus withdrawal)
                plot_data["NetContribution"] = (
                    plot_data["Contribution"] - plot_data["Withdrawal"]
                )

                # Plot net contribution line
                fig.add_trace(
                    go.Scatter(
                        x=plot_data["TurnNumber"],
                        y=plot_data["NetContribution"],
                        name=f'T{team_id}_{agent_info["TeamID"][:4]}_{agent_id[:8]}',
                        mode="lines",
                        line=dict(color=team_color),
                    )
                )

                # Add skull emoji at death point if agent dies
                if death_point is not None:
                    net_at_death = (
                        death_point["Contribution"] - death_point["Withdrawal"]
                    )
                    fig.add_trace(
                        go.Scatter(
                            x=[death_point["TurnNumber"]],
                            y=[net_at_death],
                            mode="text",
                            text=["💀"],
                            textfont=dict(size=20),
                            showlegend=False,
                            hoverinfo="skip",
                        )
                    )

    # Add a zero line for reference
    fig.add_hline(y=0, line_dash="dash", line_color="gray", opacity=0.5)

    fig.update_layout(
        title="Individual Agent Net Contributions Over Time",
        xaxis_title="Turn Number",
        yaxis_title="Net Contribution (Contribution - Withdrawal)",
        hovermode="x unified",
        showlegend=True,
        legend=dict(
            groupclick="toggleitem", yanchor="top", y=0.99, xanchor="left", x=1.05
        ),
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        margin=dict(t=150, r=150, b=50, l=50),
    )

    return fig


# Callback for Team 1 visualization
@app.callback(Output("team-1", "figure"), [Input("iteration-filter", "value")])
def update_team_1(iteration):
    # Filter data for specified iteration and Team 1 agents
    filtered_data = data["agent_records"][
        (data["agent_records"]["IterationNumber"] == iteration) &
        (data["agent_records"]["TrueSomasTeamID"] == 1)
    ]

    fig = go.Figure()

    # Group by agent ID to plot each agent's rank over time
    for agent_id in filtered_data["AgentID"].unique():
        agent_data = filtered_data[filtered_data["AgentID"] == agent_id]

        # Extract rank numbers only when "Rank" appears in specialNote
        ranks = []
        turn_numbers = []
        for idx, note in enumerate(agent_data["SpecialNote"]):
            if isinstance(note, str) and "Rank:" in note:
                try:
                    rank = int(note.split("Rank: ")[1])
                    ranks.append(rank)
                    turn_numbers.append(agent_data["TurnNumber"].iloc[idx])
                except:
                    continue

        # Only add trace if ranks were found
        if ranks:
            fig.add_trace(
                go.Scatter(
                    x=turn_numbers,
                    y=ranks,
                    name=f"Agent {agent_id[:8]} Rank",
                    mode="lines+markers",
                )
            )

    # Update layout
    fig.update_layout(
        title="Team 1: Agent Ranks Over Time",
        xaxis_title="Turn Number",
        yaxis_title="Rank",
        yaxis=dict(
            type="linear",
            dtick=1
        ),
        hovermode="x unified",
        showlegend=True,
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        margin=dict(t=150, r=150, b=50, l=50),
    )

    return fig


# Add new callback
@app.callback(
    Output("contribution-plot", "figure"),
    [Input("iteration-filter", "value"),
     Input("team-filter", "value")]
)
def update_contribution_plot(iteration, team):
    # Filter data for specified iteration and team
    filtered_data = data["agent_records"][
        (data["agent_records"]["IterationNumber"] == iteration) &
        (data["agent_records"]["TrueSomasTeamID"] == team)
    ]

    fig = go.Figure()

    # Plot each agent's contribution over time
    for agent_id in filtered_data["AgentID"].unique():
        agent_data = filtered_data[filtered_data["AgentID"] == agent_id]

        fig.add_trace(
            go.Scatter(
                x=agent_data["TurnNumber"],
                y=agent_data["StatedContribution"],
                name=f"Agent {agent_id[:8]}",
                mode="lines+markers",
            )
        )

    # Update layout
    fig.update_layout(
        title=f"Team {team}: Agent Stated Contributions Over Time",
        xaxis_title="Turn Number",
        yaxis_title="Stated Contribution",
        hovermode="x unified",
        showlegend=True,
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        margin=dict(t=150, r=150, b=50, l=50),
    )

    return fig

@app.callback(
    Output("boundaries-plot", "figure"),
    [Input("iteration-filter", "value"),
     Input("team-boundaries-filter", "value"),
     Input("interval-component", "n_intervals")]
)
def update_boundaries_plot(iteration, team_id, n_intervals):
    data = load_data()
    boundaries_data = data["team1_boundaries"]
    
    # Filter data for the selected iteration and team_id
    filtered = boundaries_data[
        # (boundaries_data["IterationNumber"] == iteration) &
        (boundaries_data["TeamID"] == team_id)
    ]
    
    fig = go.Figure()
    
    # Identify boundary columns
    boundary_columns = [col for col in filtered.columns if col.startswith("Boundaries")]
    
    # Plot each boundary line
    for col in boundary_columns:
        fig.add_trace(
            go.Scatter(
                x=filtered["TurnNumber"],
                y=filtered[col],
                name=col,
                mode="lines+markers"
            )
        )
    
    fig.update_layout(
        title=f"Rank Boundaries for {team_id[:8]} - Iteration {iteration}",
        xaxis_title="Turn Number",
        yaxis_title="Rank Boundary",
        hovermode="x unified",
        showlegend=True,
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        margin=dict(t=150, r=150, b=50, l=50)
    )
    
    return fig


# Note: Don't forget to add corresponding div in the layout:
# html.Div([
#     html.H3("Team 1"),
#     dcc.Graph(id="team-1")
# ])

# Add some CSS styling
app.index_string = """
<!DOCTYPE html>
<html>
    <head>
        <title>Agent Performance Dashboard</title>
        <style>
            .graph-container {
                width: 100%;
                margin: 20px;
                padding: 20px;
                box-shadow: 0 0 10px rgba(0,0,0,0.1);
                border-radius: 5px;
                display: flex;
                justify-content: center;
            }
            h1 {
                text-align: center;
                color: #2c3e50;
                padding: 20px;
            }
            h3 {
                color: #34495e;
            }
        </style>
    </head>
    <body>
        {%app_entry%}
        <footer>
            {%config%}
            {%scripts%}
            {%renderer%}
        </footer>
    </body>
</html>
"""

if __name__ == "__main__":
    app.run_server(debug=True)
