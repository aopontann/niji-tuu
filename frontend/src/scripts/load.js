import { initializeApp } from "firebase/app";
import { firebaseConfig } from "./main";

const modalLoadingEle = document.getElementById("modal-loading");

initializeApp(firebaseConfig);

window.onload = async () => {
  console.log("load");
  modalLoadingEle.classList.add("is-active");

  // 既に購買済みか
  if (Notification.permission !== "granted") {
    console.log("not granted")
    modalLoadingEle.classList.remove("is-active");
    return
  }
};
