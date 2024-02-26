var chatDiv = document.getElementById("chatLog");
var chatInput = document.getElementById("chatInput");
var chatButton = document.getElementById("chatButton");

console.log("establishing websocket");
WS = new WebSocket('wss://127.0.0.1/websocket/');
WS.onopen = (event) => {
    console.log("onopen event: ", event);
    WS.send('{"msg":"entered chat"}');
};

WS.onerror = (event) => {
    console.log("onerror event: ", event);
};

WS.onmessage = (event) => {
    console.log("onmessage event: ", event);
    msgData = JSON.parse(event.data);
    msg = document.createElement("p");
    msg.textContent = `${msgData.timestamp}[${msgData.userName}]:${msgData.msg}`;
    chatDiv.appendChild(msg);
};

WS.onclose = (event) => {
    console.log("onclose event: ", event);
    msg = document.createElement("p");
    msg.textContent = `connection lost`;
    chatDiv.appendChild(msg);
};

function sendMessage(event) {
    if (event.inputType === "insertLineBreak" || event.type === "click") {
        let msgText = chatInput.value;
        WS.send(`{"msg":"${msgText}"}`);
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