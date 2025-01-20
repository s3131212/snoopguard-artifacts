# LINE Chatbot E2EE and Hide Sender Analysis

This repository contains the script referenced in Section 4.2, "Platforms with Chatbot Support", of our research. The script is designed to construct a basic chatbot. Additionally, the HTTP messages generated during the script's execution have been collected and are included.

### 1. `index.js`
This JavaScript-based script is designed to configure a server that listens for incoming webhooks from the LINE server. It retrieves the latest messages and responds to them accordingly.

### 2. `http.txt`
The file contain a HTTP message intercepted during communication with the LINE server. The data was collected using Wireshark and provides evidence of the following observations in Section 4.2:

- **Group E2EE with chatbots**: Messages are transmitted securely via TLS; however, the content remains unencrypted beyond transport-level protection.
- **Hide sender**: Chatbots are capable of accessing the sender's User ID along with the message content.

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
   Install Wireshark from [Wireshark · Download](https://www.wireshark.org/download.html)
4. **ngrok**  
   1. Install ngrok from [Download ngrok](https://download.ngrok.com/downloads/linux)
   2. execute command `ngrok http 3000` to get a public url.
### Register Chatbot

1. Navigate to [LINE Developers](https://developers.line.biz/console/) and create a new provider.  
2. Within the newly created provider, create a new channel using the Messaging API.  
3. Access the **Basic Settings** of the specific channel in [LINE Developers](https://developers.line.biz/console/). Copy the **Channel Secret** and populate the `channelSecret` variable in `index.js`.  
4. Go to the **Messaging API** section of the specific channel:  
   1. Copy the **Channel Access Token** and fill in the `channelAccessToken` variable in `index.js`.  
   2. In the **Webhook URL** field, input your ngrok public URL.  
   3. Enable the **Use Webhook** option.  
5. Create a LINE group that includes both human participants and the chatbot. This group will be used as the experimental environment to observe chatbot interactions under the platform's privacy settings.

### Launch Chatbot  

Follow these steps to execute the chatbot and capture network traffic:  

1. **Start Wireshark**  
   Launch the Wireshark GUI.
2. **Set Up TLS Key Logging**  
   Specify the `(Pre)-Master-Secret log filename` in Wireshark’s preferences, as outlined in the official documentation: [TLS - Wireshark Wiki](https://wiki.wireshark.org/TLS#preference-settings).  
3. **Start Packet Capture**  
   Select the loopback network interface for ngrok traffic and begin capturing packets.
4. **Run the Chatbot**  
   In the root directory of this repository, execute the chatbot script with the following command:  
   ```bash  
   node --tls-keylog="/tmp/keylog.txt" index.js  
   ```  
   - Ensure the `--tls-keylog` flag points to the same filename specified in Step 2.  
   - For more details on the `--tls-keylog` flag, refer to the Node.js documentation: [Command-line API | Node.js v20.18.0 Documentation](https://nodejs.org/download/release/v20.18.0/docs/api/cli.html#--tls-keylogfile).  
5. **Start forwarding**
   Execute the following command to have a public endpoint forward HTTP request to local server.
   ```bash
   ngrok http 3000
   ```
6. **Interact with the Chatbot**  
   Send a message within a LINE group to trigger chatbot functionality.  
7. **Filter Network Traffic**  
   In the Wireshark interface, apply the following filter to isolate HTTP messages:  
   ```text  
   _ws.col.protocol == "HTTP"  
   ```  