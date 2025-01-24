# Pushshift Telegram Analysis

This repository contains the data processing and analysis scripts for Section 3.2, "Case Study 2: Cross-Group Identification," of our research. The study investigates the prevalence and impact of chatbots' cross-group user identification in messaging platforms.

## Usage

### Data Processing
For reproducibility, the data has been exported to pickle files stored in `analysis/*.pickle`, and error logs are maintained in `import_data/errors.log`, so there is no need to re-process the data.

The steps below outline how to replicate the data export:

1. Download the [dataset](https://zenodo.org/records/3607497) and extract it into the `import_data/` directory.
2. Set up a PostgreSQL database using the schema provided in `import_data/schema.sql`.
3. Use the `import_data/import*.py` scripts to import the dataset into the PostgreSQL database. The `import_message_threading.py` script should be executed last due to dependency.
4. Execute `analysis/dump_data_to_pickle.py` to generate pickle files for further analysis.

### Analyze Data
To analyze the data and get the results presented in the paper:
1. Run `analysis/chatbot_analysis.py` to analyze *Prevalence of Cross-Group Chatbots*.
2. Run `analysis/bot_user_encounter_ana.py` to analyze *User-Chatbot Encounters*.
