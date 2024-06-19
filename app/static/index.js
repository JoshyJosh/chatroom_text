var chatLogDiv = document.getElementById("chatLogDiv");
var chatInput = document.getElementById("chatInput");
var chatButton = document.getElementById("chatButton");
var chatroomSelectList = document.getElementById("chatroomSelectList");
var chatroomCreateInput = document.getElementById("chatroomCreateInput");
var chatroomCreateButton = document.getElementById("chatroomCreateButton");
var currentChatNameTitle = document.getElementById("currentChatNameTitle");
var userListUl = document.getElementById("userSelectList");

var chatroomMap = {};
var currentChatroomID = "";

console.log("establishing websocket");
// @todo render url via template.
WS = new WebSocket('wss://127.0.0.1/websocket/');
WS.onopen = (event) => {
    console.log("onopen event: ", event);
};

WS.onerror = (event) => {
    console.log("onerror event: ", event);
};

WS.onmessage = (event) => {
    console.log("onmessage event: ", event);
    let msgData = JSON.parse(event.data);
    if (msgData.hasOwnProperty("text")) {
        msgP = document.createElement("p");
        msgP.textContent = `${msgData.text.timestamp}[${msgData.text.userName}]:${msgData.text.msg}`;
 
        chatroomMap[msgData.text.chatroomID].logs.push(msgP.textContent);

        appendChatLogDOM(msgData.text.chatroomID, msgData);
    } else if (msgData.hasOwnProperty("chatroom")) {
        if (msgData.chatroom.hasOwnProperty("enter")) {
            let chatroomID = msgData.chatroom.enter.chatroomID; 
            chatroomMap[chatroomID] = {
                name: msgData.chatroom.enter.chatroomName,
                logs: []
            };

            let chatroomLi = document.createElement("li"); 
            chatroomLi.setAttribute("chatroomid", chatroomID);
            chatroomLi.className = "chatroomRosterEntry";

            let chatroomP = document.createElement("p");
            chatroomP.textContent = msgData.chatroom.enter.chatroomName;
            chatroomP.addEventListener("click", selectChatroomBtn);
            chatroomLi.appendChild(chatroomP);

            if (msgData.chatroom.enter.chatroomName !== "mainChat") {
                let chatroomDeleteBtn = document.createElement("button");
                chatroomDeleteBtn.textContent = "Delete";
                chatroomDeleteBtn.addEventListener("click", deleteChatroomBtn);
                chatroomLi.appendChild(chatroomDeleteBtn);

                let chatroomUpdateBtn = document.createElement("button");
                chatroomUpdateBtn.textContent = "Rename";
                chatroomUpdateBtn.addEventListener("click", updateChatroomBtn);
                chatroomLi.appendChild(chatroomUpdateBtn);
            }

            chatroomSelectList.appendChild(chatroomLi);

            selectChatroom(chatroomID)
            listUsers(msgData.chatroom.enter.usersList)
        } else if (msgData.chatroom.hasOwnProperty("delete")) {
            let chatroomID = msgData.chatroom.delete.chatroomID; 
            delete chatroomMap[chatroomID];

            // @todo make better traversal method
            for (let i = 0; i < chatroomSelectList.childNodes.length; i++) {
                let childNode = chatroomSelectList.childNodes[i];

                if (childNode.hasAttribute("chatroomid") && childNode.getAttribute("chatroomid") === chatroomID) {
                    chatroomSelectList.removeChild(childNode);
                }
            }

            if (currentChatroomID === chatroomID) {
                currentChatNameTitle.innerText += " (archived)";
            }
        } else if (msgData.chatroom.hasOwnProperty("update")) {
            let chatroomID = msgData.chatroom.update.chatroomID;
            let newChatroomName = msgData.chatroom.update.newChatroomName;
            chatroomMap[chatroomID].name = newChatroomName;

            // @todo make better traversal method
            for (let i = 0; i < chatroomSelectList.childNodes.length; i++) {
                let childNode = chatroomSelectList.childNodes[i];

                if (childNode.hasAttribute("chatroomid") && childNode.getAttribute("chatroomid") === chatroomID) {
                    childNode.querySelector("p").innerText = newChatroomName;
                }
            }
        } else if (msgData.chatroom.hasOwnProperty("addUser")) {
            let user = msgData.chatroom.addUser;
            appendUserToList(user);
        } else if (msgData.chatroom.hasOwnProperty("removeUser")) {
            // @todo implement remove user
            console.log("not implemented");
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
        if (chatroomName === "") {
            // @todo make better error propagation
            alert("cannot set empty chatroomName");
            return;
        }

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

function appendChatLogDOM(chatroomID, msgData) {
    for (let i = chatLogDiv.childNodes.length; i < chatroomMap[chatroomID].logs.length; i++) {
        msgP = document.createElement("p");
        msgP.textContent = `${msgData.text.timestamp}[${msgData.text.userName}]:${msgData.text.msg}`;
        chatLogDiv.appendChild(msgP);       
    }
}

function selectChatroomBtn (event) {
    let chatroomID = event.originalTarget.parentNode.getAttribute("chatroomid");

    selectChatroom(chatroomID);
}

function selectChatroom (chatroomID) {
    if (currentChatroomID === chatroomID) {
        return;
    }

    currentChatNameTitle.innerText = chatroomMap[chatroomID].name;

    generateChatLogDOM(chatroomID);
    currentChatroomID = chatroomID;
}

function deleteChatroomBtn (event) {
    let chatroomID = event.originalTarget.parentNode.getAttribute("chatroomid");

    deleteChatroom(chatroomID);
}

function deleteChatroom (chatroomID) {
    WS.send(`{"chatroom":{"delete":{"chatroomID":"${chatroomID}"}}}`);
}

function updateChatroomBtn(event) {
    var chatroomID = event.originalTarget.parentNode.getAttribute("chatroomid");

    // Create form element
    var form = document.createElement('form');
    form.classList.add('popup');
    
    // Create input element
    var input = document.createElement('input');
    input.type = 'text';
    input.placeholder = 'Enter new chatroom name';
    
    // Create submit button
    var submitButton = document.createElement('button');
    submitButton.type = 'submit';
    submitButton.textContent = 'Submit';
    
    // Append input and submit button to form
    form.appendChild(input);
    form.appendChild(submitButton);
    
    // Append form to body
    document.body.appendChild(form);
    
    // Show the popup
    form.style.display = 'block';

    // Focus to the text input
    input.focus();
    
    // Prevent default form submission behavior
    form.addEventListener('submit', function(event) {
      event.preventDefault();
      let newChatroomName = input.value;

      WS.send(`{"chatroom":{"update":{"chatroomID":"${chatroomID}","newChatroomName":"${newChatroomName}"}}}`);
      form.remove(); // Remove the popup after submission
    });
}

// @todo list this on enter and rerender on new chatroom.
function listUsers(users) {
    userListUl.textContent = "";
    for (let i = 0; i < users.length; i++) {
        appendUserToList(users[i])
    }
}

function appendUserToList(user) {
    let userLi = document.createElement("li"); 
    userLi.setAttribute("userid", user.id);
    userLi.className = "userListEntry";
    userLi.innerText = user.name;

    userListUl.appendChild(userLi);
}