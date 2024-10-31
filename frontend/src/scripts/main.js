import { getMessaging, getToken } from "firebase/messaging";
import { toast } from "bulma-toast";

const modalLoadingEle = document.getElementById("modal-loading");

export async function fcmToken(method, token) {
  console.log("token:", token);
  return await fetch("/api/subscription", {
    method,
    cache: "no-cache",
    headers: { "Authorization": `Bearer: ${token}` },
  });
}

export const isIos = () => {
  const userAgent = window.navigator.userAgent.toLowerCase();
  return /iphone|ipad|ipod/.test(userAgent);
};

export const isInStandaloneMode = () => {
  return window.matchMedia("(display-mode: standalone)").matches;
};

export const firebaseConfig = {
  apiKey: "AIzaSyBzGkZMCgRMfLa5MOPMDmycQT_Jb3wTQp8",
  authDomain: "niji-tuu.firebaseapp.com",
  projectId: "niji-tuu",
  storageBucket: "niji-tuu.appspot.com",
  messagingSenderId: "243582453217",
  appId: "1:243582453217:web:3c716c9d91edc5a1037ea0",
};

export const vapidKey =
  "BCSvj0H4g72CXuyK_CUy2oygQyRXDyX_BaR2ACtfmEYm2jLj-qCymSnDhfp7acuBISkKxj_UC1TKd6eOPcfr27w";

export const getFCMToken = async () => {
  const messaging = getMessaging();
  try {
    const currentToken = await getToken(messaging, { vapidKey });
    console.log("generated token:", currentToken);
    return currentToken;
  } catch (error) {
    // 通知権限がブロックされている場合
    if (Notification.permission === "denied") {
      window.alert("通知がブロックされています。");
    }

    // 通知権限がブロックされていないが、ユーザーの許可を得れていない場合
    if (Notification.permission === "default") {
      window.alert("通知を許可してください。");
    }

    if (Notification.permission === "granted") {
      return await getToken(messaging, { vapidKey });
    }
  }
  return "";
};

export const createTagElement = (topic, currentToken) => {
  const controlEle = document.createElement("div")
  controlEle.id = topic.ID
  controlEle.className = "control"

  const tagEle = document.createElement("div")
  tagEle.className = "tags has-addons"

  const textEle = document.createElement("div")
  textEle.className = "tag"
  textEle.innerText = topic.Name

  const deleteEle = document.createElement("button")
  deleteEle.className = "tag is-delete"

  deleteEle.addEventListener("click", async () => {
    modalLoadingEle.classList.add("is-active");
    const res = await fetch(`${import.meta.env.PUBLIC_BASE_URL}/api/topic`, {
      method: "DELETE",
      cache: "no-cache",
      headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer: ${currentToken}`,
      },
      body: JSON.stringify({ topic_id: topic.ID }), // 本体のデータ型は "Content-Type" ヘッダーと一致させる必要があります
    });
    if (res.ok) {
      document.getElementById(topic.ID).remove()
      toast({
        message: "登録成功",
        type: "is-success",
        pauseOnHover: true,
        opacity: 5,
        extraClasses: "mt-6"
      });
    }
    modalLoadingEle.classList.remove("is-active");
  })

  tagEle.appendChild(textEle)
  tagEle.appendChild(deleteEle)

  controlEle.appendChild(tagEle)
  return controlEle
}