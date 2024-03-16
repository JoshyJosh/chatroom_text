var chatLogDiv = document.getElementById("chatLogDiv");
var chatInput = document.getElementById("chatInput");
var chatButton = document.getElementById("chatButton");
var chatroomSelectDiv = document.getElementById("chatroomSelectDiv");
var chatroomCreateInput = document.getElementById("chatroomCreateInput");
var chatroomCreateButton = document.getElementById("chatroomCreateButton");
var currentChatNameTitle = document.getElementById("currentChatNameTitle");

var chatroomMap = {};
var currentChatroomID = "";

console.log("establishing websocket");
WS = new WebSocket('wss://127.0.0.1/websocket/');
WS.onopen = (event) => {
    console.log("onopen event: ", event);
};

WS.onerror = (event) => {
    console.log("onerror event: ", event);
};

WS.onmessage = (event) => {
    console.log("onmessage event: ", event);
    msgData = JSON.parse(event.data);
    if (msgData.hasOwnProperty("text")) {
        msgP = document.createElement("p");
        msgP.textContent = `${msgData.text.timestamp}[${msgData.text.userName}]:${msgData.text.msg}`;
 
        chatroomMap[msgData.text.chatroomID].logs.push(msgP.textContent);

        appendChatLogDOM(msgData.text.chatroomID);
    } else if (msgData.hasOwnProperty("chatroom")) {
        if (msgData.chatroom.hasOwnProperty("enter")) {
            let chatroomID = msgData.chatroom.enter.chatroomID; 
            chatroomMap[chatroomID] = {
                name: msgData.chatroom.enter.chatroomName,
                logs: []
            };

            chatroomP = document.createElement("p");
            chatroomP.textContent = msgData.chatroom.enter.chatroomName;
            chatroomP.setAttribute("chatroomID", chatroomID);
            chatroomP.className = "chatroomRosterEntry";
            chatroomP.addEventListener("click", selectChatroomEntryBtn);
            chatroomSelectDiv.appendChild(chatroomP);

            selectChatroomEntry(chatroomID)
        }
    }
};

WS.onclose = (event) => {
    console.log("onclose event: ", event);
    msg = document.createElement("p");
    msg.textContent = `connection lost`;
    chatLogDiv.appendChild(msg);
};

function sendMessage(event) {
    if ((event.inputType === "insertLineBreak" && event.originalTarget === chatInput) || event.type === "click") {
        let msgText = chatInput.value;
        WS.send(`{"text":{"msg":"${msgText}","chatroomID":"${currentChatroomID}"}}`);
        chatInput.value = "";

        if (event.type === "click") {
            chatInput.focus();
        }
    }
};

chatInput.addEventListener ("beforeinput",function(event) {
    sendMessage(event);
});

chatButton.addEventListener ("click",function(event) {
    sendMessage(event);
});

chatInput.focus();

function createChatMessage(event) {
    if ((event.inputType === "insertLineBreak" && event.originalTarget === chatroomCreateInput) || event.type === "click") {
        let chatroomName = chatroomCreateInput.value;
        WS.send(`{"chatroom":{"create":{"chatroomName":"${chatroomName}"}}}`);
        chatroomCreateInput.value = "";

        if (event.type === "click") {
            chatInput.focus();
        }
    }
};

chatroomCreateInput.addEventListener ("beforeinput",function(event) {
    createChatMessage(event);
});

chatroomCreateButton.addEventListener ("click",function(event) {
    createChatMessage(event);
});

function generateChatLogDOM(chatroomID) {
    // Remove all log item paragraphs.
    while (chatLogDiv.firstChild) {
        chatLogDiv.removeChild(chatLogDiv.firstChild);
    }
    
    chatroomMap[chatroomID].logs.map(function(logElement) {
        msgP = document.createElement("p");
        msgP.textContent = logElement;
        chatLogDiv.appendChild(msgP);
    });
}

function appendChatLogDOM(chatroomID) {
    for (let i = chatLogDiv.childNodes.length; i < chatroomMap[chatroomID].logs.length; i++) {
        msgP = document.createElement("p");
        msgP.textContent = `${msgData.text.timestamp}[${msgData.text.userName}]:${msgData.text.msg}`;
        chatLogDiv.appendChild(msgP);       
    }
}

function selectChatroomEntryBtn (event) {
    let chatroomID = event.originalTarget.getAttribute("chatroomid");

    selectChatroomEntry(chatroomID);
}

function selectChatroomEntry (chatroomID) {
    if (currentChatroomID === chatroomID) {
        return;
    }

    currentChatNameTitle.innerText = chatroomMap[chatroomID].name;

    generateChatLogDOM(chatroomID);
    currentChatroomID = chatroomID;
}