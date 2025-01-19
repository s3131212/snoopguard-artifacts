from pprint import pprint

# KEY_LINE = "Send Server Side"
# FILE = "../benchmark_results/native_roundtrip/benchmark_serverside.txt"

KEY_LINE = "Send MLS"
FILE = "../benchmark_results/native_roundtrip/benchmark_MLS.txt"


# Correct the function to properly parse the number of chatbots, handling both singular and plural forms
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

with open(FILE, "r") as file:
    log_file_content = file.readlines()

parsed_results = parse_log_file(log_file_content)

pprint(parsed_results)

data_for_plot = {"No Anonymity": {}, "IGA": {}, "Pseudonymity": {}}

for result in parsed_results:
    setting = result['setting']
    chatbots = result['chatbots']
    median_time_ms = result['median_time_ms']
    if chatbots not in data_for_plot[setting]:
        data_for_plot[setting][chatbots] = []
    if result['hide_trigger'] != "No":
        continue
    data_for_plot[setting][chatbots].append(median_time_ms)

# Average the median times for each number of chatbots (if needed) and sort the keys (number of chatbots)
for setting in data_for_plot:
    for chatbots in data_for_plot[setting]:
        data_for_plot[setting][chatbots] = sum(data_for_plot[setting][chatbots]) / len(data_for_plot[setting][chatbots])
    data_for_plot[setting] = dict(sorted(data_for_plot[setting].items()))

# Generate the LaTeX code for the tikzpicture environment to plot the line chart
latex_code = """
\\begin{tikzpicture}
\\begin{axis}[
    title={Median Execution Time by Setting},
    xlabel={Number of Chatbots},
    ylabel={Median Time (ms)},
    xmin=0, xmax=50,
    ymin=0, ymax=10,
    xtick={0,10,20,30,40,50},
    ytick={0,2,4,6,8,10},
    legend pos=north west,
    ymajorgrids=true,
    grid style=dashed,
]

"""

# Add lines for each setting
for setting in ["No Anonymity", "IGA", "Pseudonymity"]:
    latex_code += "\\addplot[\n    mark=*,\n    smooth\n] coordinates {\n"
    for chatbots, median_time_ms in data_for_plot[setting].items():
        latex_code += f"    ({chatbots},{median_time_ms})\n"
    latex_code += "};\n"
    latex_code += f"\\addlegendentry{{{setting}}}\n"

latex_code += """
\\end{axis}
\\end{tikzpicture}
"""

print(latex_code)