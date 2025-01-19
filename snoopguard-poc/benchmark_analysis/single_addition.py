import os
import re
from pprint import pprint

# KEY_LINE = "==   Chatbot Addition"
# UNIT = "ms"  # Unit for the median time values

KEY_LINE = "MLS"
UNIT = "ms"  # Unit for the median time values

# iterate directories in ../benchmark_results cpu_.*
benchmark_dir = "../benchmark_results"
pattern = re.compile(r'cpu_.*')
dir_list = []

for root, dirs, files in os.walk(benchmark_dir):
    for dir_name in dirs:
        if pattern.match(dir_name):
            dir_list.append(dir_name)

# Sort the list
dir_list.sort(key=lambda x: float(x.split('_')[1]))

pprint(dir_list)

# get the benchmark results from the directories' geekbench_results.txt files
benchmark_results = {}
for dir_name in dir_list:
    file_path = os.path.join(benchmark_dir, dir_name, "geekbench_results.txt")
    with open(file_path, 'r') as file:
        lines = file.readlines()
        for line in lines:
            if "Single-Core Score" in line:
                single_core_score = int(line.split("=")[1].strip())
        benchmark_results[dir_name] = single_core_score

pprint(benchmark_results)

# Read the benchmark for chatbot addition.
def analyze_log_file(file_path):
    results = {}
    with open(file_path, 'r') as file:
        current_setting = None
        current_members = 0
        for line in file:
            # Identify and process only relevant lines
            if "Chatbot Addition" in line and KEY_LINE in line:
                parts = line.split(":")
                details = [s.strip().split(" ")[0] for s in parts[1].split(",")]
                group_members = int(details[0].strip().split(" ")[0])
                setting = "None"  # Default setting
                if "IGA" in details:
                    setting = "IGA"
                if "Pseudo" in details:
                    setting = "Pseudonymity"
                current_setting = setting
                current_members = group_members
            elif "p50" in line:
                # Extract median time
                median_time_info = line.split(";")[2].strip()
                median_time_value = float(median_time_info.split(" ")[1].replace("µs", "").replace("ms", "").replace("s", ""))
                # Convert all times to milliseconds for consistency
                if "µs" in median_time_info:
                    median_time_value /= 1000  # Convert microseconds to milliseconds
                if "s" in median_time_info and "ms" not in median_time_info and "µs" not in median_time_info:
                    median_time_value *= 1000 # Convert seconds to milliseconds
                
                # change unit
                if UNIT == "s":
                    median_time_value /= 1000

                # Save the extracted information
                if current_setting and current_members:
                    key = (current_setting, current_members)
                    results[key] = median_time_value
                
                # Reset for the next entry
                current_setting = None
                current_members = 0

    return results

# for each dir, analyze the log file and store the results
all_results = {}
for dir_name in dir_list:
    file_path = os.path.join(benchmark_dir, dir_name, "benchmark_add.txt")
    results = analyze_log_file(file_path)
    all_results[dir_name] = results

pprint(all_results)
# For each dir, get the results for No anonymitu, IGA and Pseudonymity when # member = 30
filtered_results = {}
for dir_name in dir_list:
    filtered_results[dir_name] = {}
    for setting in ["None", "IGA", "Pseudonymity"]:
        filtered_results[dir_name][setting] = all_results[dir_name][(setting, 20)]

pprint(filtered_results)

# Generate the LaTeX code for the tikzpicture environment to plot the line chart. The x-axis is the geekbench score. The y-axis is the median time. There are three lines, one for each setting.
latex_code = ""

# Add lines for each setting
for setting in ["None", "IGA", "Pseudonymity"]:
    latex_code += "\\addplot[smooth,mark=*,blue] plot coordinates {"
    for dir_name in dir_list:
        geekbench_score = benchmark_results[dir_name]
        median_time = filtered_results[dir_name][setting]
        latex_code += f"({geekbench_score},{median_time}) "
    latex_code += "};\n"
    latex_code += f"\\addlegendentry{{{setting}}}\n"

latex_code += ""

print(latex_code)