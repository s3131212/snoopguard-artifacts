# Discord Chatbot E2EE and Hide Sender Analysis

This repository contains the scripts referenced in Section 4.2, "Platforms with Chatbot Support", of our research. The scripts are designed to construct basic chatbots with varying permission configurations. Additionally, the WebSocket messages generated during the execution of these scripts have been collected and are included.

`command-only.js`, `with-message-content.js`, and `without-message-content.js` are scripts designed to serve as three Discord chatbots. The permissions of the three corresponding bots differ from each other. Each script is accompanied by the corresponding WebSocket message intercepted during communication with the Discord server. The data was collected using Wireshark and provides evidence of the following observations in Section 4.2:

- **Group E2EE with chatbots**: Messages are transmitted securely via TLS; however, the content remains unencrypted beyond transport-level protection.
- **Hide sender**: Chatbots are capable of accessing the sender's User ID when recieving message.

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

### Register Chatbots  

1. **Create a Discord Server**  
   - Follow the instructions provided here: [How do I create a server? – Discord](https://support.discord.com/hc/en-us/articles/204849977-How-do-I-create-a-server).  

2. **Create Chatbots**  
   - Access the [Discord Developer Portal — My Applications](https://discord.com/developers/applications) and create new applications for the chatbots.  

3. **Enable Message Content Intent** (for `with-message-content.js`)  
   - Navigate to **Bot** > **Privileged Gateway Intents** for the selected application and enable the **Message Content Intent** option.  

4. **Configure Bot Token**  
   - Navigate to **Bot** > **Build-A-Bot**, click **Reset Token**, and copy the generated token. Use this token as the parameter for the `client.login` function in the three scripts.  

5. **Install Bots to Your Discord Server**  
   - Navigate to **Installation** for the selected application.  
     - Under **Installation Contexts**, select **Guild Install** only.  
     - Under **Default Install Settings**:  
       - Select **Send Messages** in the **permissions** field.
       - For `with-message-content.js` and `without-message-content.js`, select **bot** in the **scopes** field.  
       - For `command-only.js`, select **applications.commands** in the **scopes** field.  
     - Under **Install Link**, copy the generated link and use it to install the bot to your Discord server.  

6. **Register a Command** (for `command-only.js`)  
   - Populate the following variables in `register.js`:  
     - **clientId**: Found under **General Information** > **Application ID** in the [Discord Developer Portal](https://discord.com/developers/applications) > Selected App.  
     - **guildId**: Located in the URL of your Discord server when accessed via the web client.  
     - **token**: Same as the bot token retrieved in Step 4.  
   - Run the following command to register the bot commands:  
     ```bash  
     node register.js  
     ```

### Launch Chatbot  

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
   node --tls-keylog="/tmp/keylog.txt" {command-only,with-message-content,without-message-content}.js  
   ```  
   - Ensure the `--tls-keylog` flag points to the same filename specified in Step 2.  
   - For more details on the `--tls-keylog` flag, refer to the Node.js documentation: [Command-line API | Node.js v20.18.0 Documentation](https://nodejs.org/download/release/v20.18.0/docs/api/cli.html#--tls-keylogfile).  
5. **Interact with the Chatbot**  
   Send a message within a Discord channel to trigger chatbot functionality.  
6. **Filter Network Traffic**  
   In the Wireshark interface, apply the following filter to isolate WebSocket messages:  
   ```text  
   (websocket)
   ```  
