This is a pet project for making a text chatroom (IRC-like) in order to practice DDD structure, HTTP/2, nginx, account management (using kratos and kratos-ui), websockets and TLS.

If you're looking for a fancy frontend application, this isn't it :]
### Setup instructions
1. Run `generate.sh` in order to make certs for `127.0.0.1` or generate your own and put them in the `certs` folder (Note, custom domains are currently not supported)
2. Run `make up`
3. Go to `127.0.0.1` . You will be prompted to trust certs x3, since they're self signed and you will be redirected between chatroom -> kratos -> kratos-ui in order to create an account.
4. (On first sign in) go to `127.0.0.1:4436` in order to access mailslurper and get sign in code.
5. After the sign in you should be redirected to `127.0.0.1` and see the chatroom. Since its localhost you will be the only one present.
### Todo list
- Make an option for a variable DNS and verify its certs.
- Add option for multiple users (only one active websocket connection is alllowed by kratos client, DDOS protection)
- Allow for chatroom creation and user invitation.
- Update Makefile to generate certs if not present and start up app.