self.addEventListener("notificationclick", (event) => {
  console.log("event:", event);
  try {
    event.notification.close();
    clients.openWindow(event.notification.data.FCM_MSG.notification.click_action ?? "/");
  } catch (e) {
    // デバッグ用なので本番では消してもよいです
    console.error(e);
  }
});

importScripts("https://www.gstatic.com/firebasejs/9.2.0/firebase-app-compat.js");
importScripts("https://www.gstatic.com/firebasejs/9.2.0/firebase-messaging-compat.js");

// Initialize the Firebase app in the service worker by passing in
// your app's Firebase config object.
// https://firebase.google.com/docs/web/setup#config-object
firebase.initializeApp({
  apiKey: "AIzaSyBzGkZMCgRMfLa5MOPMDmycQT_Jb3wTQp8",
  authDomain: "niji-tuu.firebaseapp.com",
  projectId: "niji-tuu",
  storageBucket: "niji-tuu.appspot.com",
  messagingSenderId: "243582453217",
  appId: "1:243582453217:web:3c716c9d91edc5a1037ea0",
});

// Retrieve an instance of Firebase Messaging so that it can handle background
// messages.
const messaging = firebase.messaging();
