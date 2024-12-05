import pandas as pd
import dash
from dash import dcc, html
from dash.dependencies import Input, Output
import plotly.express as px
import plotly.graph_objects as go

# Add these constants at the top of the file for consistent plot styling
PLOT_HEIGHT = 600  # Increased height
PLOT_WIDTH = 1000   # Increased from 700 to 1000
PLOT_MARGIN = dict(r=150)  # More space on the right for legends

# Read the CSV files
base_path = 'visualization_output/csv_data'
agent_records = pd.read_csv(f'{base_path}/agent_records.csv')
common_records = pd.read_csv(f'{base_path}/common_records.csv')
team_records = pd.read_csv(f'{base_path}/team_records.csv')

# Initialize the Dash app
app = dash.Dash(__name__)

# Create the layout
app.layout = html.Div([
    html.H1("Agent Performance Dashboard"),
    
    html.Div([
        html.Div([
            html.H3("Score Evolution Over Time"),
            dcc.Graph(id='score-evolution'),
        ], className='graph-container'),
        
        html.Div([
            html.H3("Contribution vs Withdrawal"),
            dcc.Graph(id='contribution-withdrawal'),
        ], className='graph-container'),
        
        html.Div([
            html.H3("Common Pool Evolution"),
            dcc.Graph(id='common-pool'),
        ], className='graph-container'),
        
        html.Div([
            html.H3("Agent Status"),
            dcc.Graph(id='agent-status'),
        ], className='graph-container'),
    ]),
    
    # Add filters
    html.Div([
        html.H3("Filters"),
        dcc.Dropdown(
            id='iteration-filter',
            options=[{'label': f'Iteration {i}', 'value': i} 
                    for i in agent_records['IterationNumber'].unique()],
            value=0,
            clearable=False
        ),
    ], style={'width': '200px', 'margin': '20px'})
])

# Callback for score evolution with threshold overlay
@app.callback(
    Output('score-evolution', 'figure'),
    [Input('iteration-filter', 'value')]
)
def update_score_evolution(iteration):
    filtered_data = agent_records[agent_records['IterationNumber'] == iteration]
    threshold_data = common_records[common_records['IterationNumber'] == iteration]
    
    fig = go.Figure()
    
    # Group agents by their true team ID
    for team_id in filtered_data['TrueSomasTeamID'].unique():
        team_data = filtered_data[filtered_data['TrueSomasTeamID'] == team_id]
        
        # Create a consistent color for this team
        team_color = px.colors.qualitative.Set1[team_id % len(px.colors.qualitative.Set1)]
        
        # Plot each agent in the team with the same color
        for agent_id in team_data['AgentID'].unique():
            agent_data = team_data[team_data['AgentID'] == agent_id]
            
            # Find where agent dies (if it does)
            death_point = agent_data[agent_data['IsAlive'] == False].iloc[0] if any(~agent_data['IsAlive']) else None
            
            # Plot including the death point
            if death_point is not None:
                # Include all data up to and including death point
                plot_data = agent_data[agent_data['TurnNumber'] <= death_point['TurnNumber']]
            else:
                plot_data = agent_data
                
            if not plot_data.empty:
                fig.add_trace(go.Scatter(
                    x=plot_data['TurnNumber'],
                    y=plot_data['Score'],
                    name=f'Team {team_id} - Agent {agent_id[:8]}',
                    mode='lines',
                    line=dict(color=team_color)
                ))
            
            # Add skull emoji at death point if agent dies
            if death_point is not None:
                fig.add_trace(go.Scatter(
                    x=[death_point['TurnNumber']],
                    y=[death_point['Score']],
                    mode='text',
                    text=['💀'],
                    textfont=dict(size=20),
                    showlegend=False,
                    hoverinfo='skip'
                ))
    
    # Add threshold line
    fig.add_trace(go.Scatter(
        x=threshold_data['TurnNumber'],
        y=threshold_data['Threshold'],
        name='Threshold',
        mode='lines',
        line=dict(color='red', dash='dash'),
    ))
    
    # Add vertical lines for threshold application turns
    threshold_turns = threshold_data[threshold_data['ThresholdAppliedInTurn'] == True]['TurnNumber']
    for turn in threshold_turns:
        fig.add_vline(
            x=turn,
            line_dash="dash",
            line_color="red",
            opacity=0.5
        )
    
    fig.update_layout(
        title='Score Evolution Over Time with Threshold',
        xaxis_title="Turn Number",
        yaxis_title="Score",
        hovermode='x unified',
        showlegend=True,
        legend=dict(
            groupclick="toggleitem",
            yanchor="top",
            y=0.99,
            xanchor="left",
            x=1.05
        ),
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        margin=PLOT_MARGIN
    )
    
    return fig

# Callback for contribution vs withdrawal
@app.callback(
    Output('contribution-withdrawal', 'figure'),
    [Input('iteration-filter', 'value')]
)
def update_contribution_withdrawal(iteration):
    filtered_data = agent_records[agent_records['IterationNumber'] == iteration]
    
    fig = go.Figure()
    
    # Group by team ID
    for team_id in filtered_data['TrueSomasTeamID'].unique():
        team_data = filtered_data[filtered_data['TrueSomasTeamID'] == team_id]
        team_color = px.colors.qualitative.Set1[team_id % len(px.colors.qualitative.Set1)]
        
        # Calculate mean values per turn for this team
        mean_data = team_data.groupby('TurnNumber').agg({
            'Contribution': 'mean',
            'Withdrawal': 'mean'
        }).reset_index()
        
        fig.add_trace(go.Scatter(
            x=mean_data['TurnNumber'],
            y=mean_data['Contribution'],
            name=f'Team {team_id} - Contribution',
            mode='lines+markers',
            line=dict(color=team_color),
            marker=dict(symbol='circle')
        ))
        
        fig.add_trace(go.Scatter(
            x=mean_data['TurnNumber'],
            y=mean_data['Withdrawal'],
            name=f'Team {team_id} - Withdrawal',
            mode='lines+markers',
            line=dict(color=team_color, dash='dash'),
            marker=dict(symbol='square')
        ))
    
    fig.update_layout(
        title='Average Contribution vs Withdrawal Over Time by Team',
        xaxis_title="Turn Number",
        yaxis_title="Value",
        hovermode='x unified',
        showlegend=True,
        legend=dict(
            groupclick="toggleitem",
            yanchor="top",
            y=0.99,
            xanchor="left",
            x=1.05
        ),
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        margin=PLOT_MARGIN
    )
    
    return fig

# Callback for common pool evolution
@app.callback(
    Output('common-pool', 'figure'),
    [Input('iteration-filter', 'value')]
)
def update_common_pool(iteration):
    filtered_data = team_records[team_records['IterationNumber'] == iteration]
    
    fig = go.Figure()
    
    for team_id in filtered_data['TeamID'].unique():
        team_data = filtered_data[filtered_data['TeamID'] == team_id]
        if team_id != '00000000-0000-0000-0000-000000000000':  # Skip empty team
            fig.add_trace(go.Scatter(
                x=team_data['TurnNumber'],
                y=team_data['TeamCommonPool'],
                name=f'Team {team_id[:8]}',
                mode='lines+markers'
            ))
    
    fig.update_layout(
        title='Team Common Pool Evolution',
        xaxis_title="Turn Number",
        yaxis_title="Common Pool Value",
        hovermode='x unified',
        showlegend=True,
        legend=dict(
            yanchor="top",
            y=0.99,
            xanchor="left",
            x=1.05
        ),
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        margin=PLOT_MARGIN
    )
    
    return fig

# Callback for agent status
@app.callback(
    Output('agent-status', 'figure'),
    [Input('iteration-filter', 'value')]
)
def update_agent_status(iteration):
    filtered_data = agent_records[agent_records['IterationNumber'] == iteration]
    
    status_counts = filtered_data.groupby(['TurnNumber', 'IsAlive']).size().unstack(fill_value=0)
    
    fig = go.Figure()
    
    fig.add_trace(go.Bar(
        x=status_counts.index,
        y=status_counts[True],
        name='Alive',
        marker_color='green'
    ))
    
    fig.add_trace(go.Bar(
        x=status_counts.index,
        y=status_counts[False],
        name='Dead',
        marker_color='red'
    ))
    
    fig.update_layout(
        title='Agent Status Over Time',
        xaxis_title="Turn Number",
        yaxis_title="Number of Agents",
        barmode='stack',
        showlegend=True,
        legend=dict(
            yanchor="top",
            y=0.99,
            xanchor="left",
            x=1.05
        ),
        height=PLOT_HEIGHT,
        width=PLOT_WIDTH,
        margin=PLOT_MARGIN
    )
        
    return fig

# Add some CSS styling
app.index_string = '''
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
'''

if __name__ == '__main__':
    app.run_server(debug=True) 