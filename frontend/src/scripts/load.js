import { initializeApp } from "firebase/app";
import { fcmToken, firebaseConfig, vapidKey, createTagElement } from "./main";
import { getMessaging, getToken, deleteToken } from "firebase/messaging";
import { toast } from "bulma-toast";

const songEle = document.getElementById("checkbox-song");
const infoEle = document.getElementById("checkbox-info");
const topicResiterListEle = document.getElementById("topic-register-list");
const topicListEle = document.getElementById("topic-list");
const modalTopicListEle = document.getElementById("modal-topic-list");
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

  const messaging = getMessaging();
  const currentToken = await getToken(messaging, { vapidKey });

  // APIサーバーから購買情報を取得　歌ってみた動画を通知する許可をしているか...
  const resSong = await fetch(`${import.meta.env.PUBLIC_BASE_URL}/api/song`, {
    method: "GET",
    headers: {
      "Authorization": `Bearer: ${currentToken}`,
    }
  });
  if (resSong.status == 200) {
    const songData = await resSong.json();
    console.log(songData)
    songEle.checked = songData.status;
    window.localStorage.setItem("checkbox-song", songData.status)
  } else {
    const msg = await resSong.text()
    console.log(msg)
    modalLoadingEle.classList.remove("is-active");
  }

  const resInfo = await fetch(`${import.meta.env.PUBLIC_BASE_URL}/api/info`, {
    method: "GET",
    headers: {
      "Authorization": `Bearer: ${currentToken}`,
    }
  });
  if (resInfo.status == 200) {
    const infoData = await resInfo.json();
    infoEle.checked = infoData.status;
    window.localStorage.setItem("checkbox-info", infoData.infoData)
  }

  // --------------------------------------- 登録済みキーワード取得 --------------------------------------- //
  const topicRegisterListInfo = await fetch(`${import.meta.env.PUBLIC_BASE_URL}/api/topic`, {
    method: "GET",
    headers: {
      "Authorization": `Bearer: ${currentToken}`,
    }
  });
  if (topicRegisterListInfo.status == 200) {
    const topicRegisterList = await topicRegisterListInfo.json();
    for (const topic of topicRegisterList) {
      const controlEle = createTagElement(topic, currentToken)
      topicResiterListEle.appendChild(controlEle)
    }
  }
  // ---------------------------------------------------------------------------------------------------- //

  // --------------------------- キーワード追加処理 --------------------------- //
  const addTopicEle = document.createElement("button")
  addTopicEle.addEventListener("click", () => {
    if (topicResiterListEle.childElementCount > 5) {
      toast({
        message: "5個以上のキーワードは追加できません",
        type: "is-warning",
        pauseOnHover: true,
        opacity: 5,
        extraClasses: "mt-6"
      });
      return
    }
    document.getElementById("modal-topic-list").classList.add("is-active")
  })
  addTopicEle.id = "add-topic"
  addTopicEle.className = "tag"
  addTopicEle.innerText = "キーワード追加"
  topicResiterListEle.appendChild(addTopicEle)
  // ----------------------------------------------------------------------- //

  // ------------------------------------- topic リスト処理 ------------------------------------- //
  const topicListInfo = await fetch(`${import.meta.env.PUBLIC_BASE_URL}/api/topic/list`, {
    method: "GET",
    headers: {
      "Authorization": `Bearer: ${currentToken}`,
    }
  });
  const topicList = await topicListInfo.json();
  console.log("topicList:", topicList)
  for (const topic of topicList) {
    const topicEle = document.createElement("button")

    topicEle.addEventListener('click', async (params) => {
      document.getElementById("modal-loading")?.classList.add("is-active");
      const res = await fetch(`${import.meta.env.PUBLIC_BASE_URL}/api/topic`, {
        method: "POST",
        headers: {
          "Authorization": `Bearer: ${currentToken}`,
        },
        body: JSON.stringify({ topic_id: topic.ID })
      });
      const controlEle = createTagElement(topic, currentToken)
      topicResiterListEle.insertBefore(controlEle, document.getElementById("add-topic"))
      document.getElementById("modal-loading")?.classList.remove("is-active");
      document.getElementById("modal-topic-list").classList.remove("is-active")
      if (res.ok) {
        toast({
          message: "登録成功",
          type: "is-success",
          pauseOnHover: true,
          opacity: 5,
          extraClasses: "mt-6"
        });
      }
    })

    topicEle.className = "tag"
    topicEle.innerText = topic.Name
    topicListEle.appendChild(topicEle)
  }
  // ------------------------------------------------------------------------------------------------- //

  const deleteEle = document.getElementById("unsubscription");
  deleteEle.addEventListener("click", async () => {
    console.log("DELETE");
    // ローディング表示
    deleteEle.classList.add("is-loading");

    const currentToken = await getToken(messaging, { vapidKey });

    // FCMトークン削除
    const deleted = await deleteToken(messaging);
    if (!deleted) {
      window.alert("トークンの削除に失敗しました");
      // ローディング解除
      deleteEle.classList.remove("is-loading");
    }
    console.log('deleted')

    const res = await fetch(`${import.meta.env.PUBLIC_BASE_URL}/api/unsubscription`, {
      method: "POST",
      cache: "no-cache",
      headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer: ${currentToken}`,
      }
    });
    if (!res.ok) {
      window.alert("トークンの削除に失敗しました");
      // ローディング解除
      deleteEle.classList.remove("is-loading");
    } else {
      while (topicResiterListEle && topicResiterListEle.childElementCount > 1) {
        topicResiterListEle.removeChild(topicResiterListEle.firstChild)
      }
    }

    // ローディング解除
    deleteEle.classList.remove("is-loading");

    toast({
      message: "削除成功",
      type: "is-success",
      // dismissible: true,
      pauseOnHover: true,
      opacity: 5,
      extraClasses: "mt-6"
    });

    // 設定初期化
    const songEle = document.getElementById("checkbox-song");
    const infoEle = document.getElementById("checkbox-info");
    songEle.checked = false;
    window.localStorage.setItem("checkbox-song", "false")
    infoEle.checked = false;
    window.localStorage.setItem("checkbox-info", "false")
  });

  console.log('complated!!!')
  modalLoadingEle.classList.remove("is-active");


  // infoEle.checked = infoData.status;
  // window.localStorage.setItem("checkbox-info", infoData.infoData)

  // const res = await fcmToken("GET", currentToken);
  // if (res.status == 204) {
  //   console.log("no content");
  //   return
  // }
  // if (!res.ok) {
  //   window.alert("購買情報の取得に失敗しました。");
  //   return;
  // }

  // const data = await res.json();
  // console.log("data", data);

  // if (data == null) {
  //   return;
  // }

  // // 最新の購買情報に応じて要素を変更
  // songEle.checked = data.song;
  // window.localStorage.setItem("checkbox-song", data.song)
  // infoEle.checked = data.info;
  // window.localStorage.setItem("checkbox-info", data.info);
};
