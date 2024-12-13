import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt

# List of team sizes and their corresponding file paths
team_sizes = [2, 4, 5, 10, 25, 50]
base_path = r'visualization_output\experiment_csv_data'

# Loop through each team size
for team_size in team_sizes:
    file_path = f'{base_path}\\{team_size}AgentTeams\\team_records.csv'
    
    # Load the CSV file
    data = pd.read_csv(file_path)
    
    # Exclude rows where TeamAoA equals 5
    filtered_data = data[data['TeamAoA'] != 5]
    
    # Group by TurnNumber and calculate the overall mean TeamSize
    average_team_size = (
        filtered_data.groupby('TurnNumber')['TeamSize']
        .mean()
        .reset_index()
    )
    
    # Set Seaborn style
    sns.set(style="whitegrid")
    
    # Create the plot
    plt.figure(figsize=(12, 8))
    sns.lineplot(
        data=average_team_size,
        x='TurnNumber',
        y='TeamSize',
        color='blue',
        linewidth=2
    )
    
    # Customise the plot
    plt.title(f'Agents Alive over Iteration with {team_size} Agent Teams', fontsize=16)
    plt.xlabel('Turn Number', fontsize=14)
    plt.ylabel('Average Team Size', fontsize=14)
    plt.grid(True, which='both', linestyle='--', linewidth=0.5)
    plt.tight_layout()
    
    # Save the plot to a file
    save_path = f'{base_path}\\{team_size}AgentTeams\\agents_alive_plot.png'
    plt.savefig(save_path)
    plt.close()  # Close the figure to avoid overlapping plots
