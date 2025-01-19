import os
import re
from pprint import pprint

# KEY_LINE = "Generate Server Side"
# FILENAME = "benchmark_serverside.txt"

KEY_LINE = "Generate MLS"
FILENAME = "benchmark_MLS.txt"

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

# Read the benchmark for message sending.
def parse_log_file(log_file_lines):
    results = []
    setting = {}
    for line in log_file_lines:
        if KEY_LINE in line:
            setting = {
                "setting": "IGA" if "IGA" in line else "Pseudonymity" if "Pseudo" in line else "No Anonymity",
                "hide_trigger": "Yes" if "Hide Trigger" in line else "No",
                "members": int(line.split(":")[1].split("members")[0].strip()),
                "chatbots": int(line.split("chatbots")[0].split(",")[-1].strip().split(" ")[0])
            }
        if "p50" in line:
            time_str = line.split("p50")[1].split(";")[0].strip()
            if "ms" in time_str:
                median_time = float(time_str.replace("ms", "").strip())  # Keep in milliseconds
            elif "µs" in time_str:
                median_time = float(time_str.replace("µs", "").strip()) / 1000  # Convert µs to milliseconds
            setting["median_time_ms"] = median_time
            results.append(setting.copy())
    return results

all_results = {}
for dir_name in dir_list:
    file_path = os.path.join(benchmark_dir, dir_name, FILENAME)
    with open(file_path, "r") as file:
        log_file_content = file.readlines()

    parsed_results = parse_log_file(log_file_content)
    all_results[dir_name] = parsed_results

pprint(all_results)

# For each dir, get the results for No anonymitu, IGA and Pseudonymity when # chatbot = 30 and no hide trigger

filtered_results = {}
for dir_name, results in all_results.items():
    filtered_results[dir_name] = {}
    for result in results:
        setting = result['setting']
        chatbots = result['chatbots']
        median_time_ms = result['median_time_ms']
        if chatbots == 30 and result['hide_trigger'] == "No":
            filtered_results[dir_name][setting] = median_time_ms

pprint(filtered_results)

# Generate the LaTeX code for the tikzpicture environment to plot the line chart. The x-axis is the geekbench score. The y-axis is the median time. There are three lines, one for each setting.

latex_code = ""

for setting in ["No Anonymity", "IGA", "Pseudonymity"]:
    latex_code += "\\addplot[smooth,mark=*,blue] plot coordinates {"
    for dir_name, times in filtered_results.items():
        geekbench_score = benchmark_results[dir_name]
        median_time = times[setting]
        latex_code += f"({geekbench_score},{median_time}) "
    latex_code += "};\n"

latex_code += ""
print(latex_code)