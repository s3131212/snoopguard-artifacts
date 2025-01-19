from pprint import pprint

# KEY_LINE = "==   Chatbot Addition"
# UNIT = "ms"  # Unit for the median time values

KEY_LINE = "MLS"
UNIT = "ms"  # Unit for the median time values

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
                print(details)
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

results = analyze_log_file("../benchmark_results/native_roundtrip/benchmark_add.txt")
pprint(results)

# Organize data for plotting
plot_data = {"None": [], "IGA": [], "Pseudonymity": []}
for (setting, members), time in results.items():
    plot_data[setting].append((members, time))

# Sort the data by number of members for consistent plotting
for setting in plot_data:
    plot_data[setting].sort()

# Add plots for each setting
latex_code = ""
colors = {"None": "color1", "IGA": "color2", "Pseudonymity": "color3"}
for setting, data in plot_data.items():
    latex_code += f"\\addplot[color={colors[setting]},mark=*] coordinates {{\n"
    for members, time in data:
        latex_code += f"({members},{time})\n"
    latex_code += "};\n"
    latex_code += f"\\addlegendentry{{{setting}}}\n"

print(latex_code)