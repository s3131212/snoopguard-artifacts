# Slack Chatbot E2EE and Hide Sender Analysis

This repository contains the scripts referenced in Section 4.2, "Platforms with Chatbot Support", of our research. The scripts are designed to construct basic chatbots with varying permission configurations. Additionally, the WebSocket messages generated during the execution of these scripts have been collected and are included.

For E2EE and hide sender properties, the `access_n_e2ee` folder includes three scripts. These scripts are tailored for bots with the following permissions: `app_mentions:read` scope, `channels:history` scope, and Slash Command functionality. Each script is accompanied by the corresponding WebSocket message intercepted during communication with the Slack server. The data was collected using Wireshark and provides evidence of the following observation in Section 4.2:

- **Group E2EE with chatbots**: Messages are transmitted securely via TLS; however, the content remains unencrypted beyond transport-level protection.


For experiments involving email retrieval based on user IDs within a group, the `email` folder contains two scripts. These are configured for bots with and without the `users:read` and `users:read.email` scopes, enabling comparative analysis of email access permission. The data provides evidence of the following observation in Section 4.2:

- **Hide sender**: Chatbots are capable of accessing the sender's within-group User ID along with the message content; however, the retrieval of user's global identifier, which is email, based User IDs require specific permissions.

## Usage

### Installation  
To replicate our analysis, ensure the following software dependencies are met:  

1. **Node.js**  
   Install the Node.js runtime from the official website: [Node.js — Download Node.js®](https://nodejs.org/en/download).  
   - **Version Used:** v20.18.0 (for consistency with this analysis).  

2. **Repository Setup**  
   Navigate to the root directory of the repository and execute the following command to install the necessary dependencies:  
   ```bash  
   npm install  
   ```  

3. **Wireshark**  
   Install Wireshark from [Wireshark · Download](https://www.wireshark.org/download.html).

To set up the Slack environment, create a Slack workspace by following the instructions here: [Create a Slack workspace | Slack](https://slack.com/help/articles/206845317-Create-a-Slack-workspace).  

### For `access_n_e2ee/` scripts: Registering Chatbots

For each bot, follow these steps:  

1. **Create a Slack Application**  
   - Visit [api.slack.com](https://api.slack.com), and create a new Slack application.

2. **For Bot with app-mentions scope**  
   - Navigate to the specific bot's **Features** > **Event Subscriptions** and create a new bot user event with the event name `app_mention`.  
   - Go to **Features** > **OAuth & Permissions** > **Scopes** > **Bot Token Scopes**, and add the `chat:write` scope.  

3. **For Bot with history scope**  
   - Navigate to **Features** > **Event Subscriptions** and create a new bot user event with the event name `message.channels`.  
   - Go to **Features** > **OAuth & Permissions** > **Scopes** > **Bot Token Scopes**, and add the `chat:write` scope.  

4. **For Bot with Slash Command Feature**  
   - Navigate to **Features** > **Slash Commands** and create a new command with the name `/echo`.  

5. **Finalize Configuration for Each Bot**  
   - Install Bot to Workspace
     - Go to **Features** > **OAuth & Permissions** > **OAuth Tokens** and click **Install to {Workspace}**.  
     - For Bot with app-mentions scope and Bot with history scope, copy the Bot User OAuth Token generated in previous step and fill in the respective script.
   - Enable Socket mode
     - Go to **Settings** > **Socket Mode** and enable socket mode.  
     - Go to **Settings** > **Basic Information** > **App-Level Tokens** and generate a token with `connections:write` scope.
     - Copy the App-Level Token generated in previous step and fill in the respective script.

### For `access_n_e2ee/` scripts: Launch Chatbot
Follow these steps to execute the chatbot and capture network traffic:

1. **Start Wireshark**  
   Launch the Wireshark GUI.
2. **Set Up TLS Key Logging**  
   Specify the `(Pre)-Master-Secret log filename` in Wireshark’s preferences, as outlined in the official documentation: [TLS - Wireshark Wiki](https://wiki.wireshark.org/TLS#preference-settings).  
3. **Start Packet Capture**  
   Select the appropriate network interface for internet traffic and begin capturing packets.
4. **Run the Chatbot**  
   In the root directory of this repository, execute the chatbot script with the following command:  
   ```bash  
   node --tls-keylog="/tmp/keylog.txt" access_n_e2ee/{app-mentions,history,slash-command}.js
   ```  
   - Ensure the `--tls-keylog` flag points to the same filename specified in Step 2.  
   - Choose one of the three script at each execution.
   - For more details on the `--tls-keylog` flag, refer to the Node.js documentation: [Command-line API | Node.js v20.18.0 Documentation](https://nodejs.org/download/release/v20.18.0/docs/api/cli.html#--tls-keylogfile).  
5. **Interact with the Chatbot**  
   Send a message within a Telegram group to trigger chatbot functionality.  
6. **Filter Network Traffic**  
   In the Wireshark interface, apply the following filter to isolate WebSocket messages:  
   ```text  
   (websocket)
   ```  

### For `email/` Scripts: Registering Chatbots  

Follow these steps for each bot:  

1. **Create a Slack Application**  
   - Visit [api.slack.com](https://api.slack.com) and create a new Slack application.  

2. **Configure Bot with Email Scopes**  
   - Navigate to **Features** > **OAuth & Permissions** > **Scopes** > **Bot Token Scopes** and add the `users:read` and `users:read.email` scopes.  

3. **Finalize Bot Configuration**  
   - Navigate to **Features** > **OAuth & Permissions** > **OAuth Tokens** and select **Install to {Workspace}**.
   - Copy the Bot User OAuth Token generated in previous step and fill in the respective script.

### For `email/` Scripts: Fetching User Emails  

1. Populate the `userId` variables in the `with-email.js` and `without-email.js` files with the desired user IDs.  

2. Execute the following command to fetch user emails:  
   ```bash  
   node email/{with-email,without-email}.js  
   ```