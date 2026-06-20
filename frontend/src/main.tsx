import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./styles.css";

function escapeHtml(value: string) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

function errorMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}

function renderBootError(error: unknown) {
  const root = document.getElementById("root");
  if (!root) {
    return;
  }
  root.innerHTML = `
    <main class="boot-error">
      <section>
        <span>Proxy-Cat</span>
        <h1>界面启动失败</h1>
        <p>${escapeHtml(errorMessage(error))}</p>
      </section>
    </main>
  `;
}

window.addEventListener("error", (event) => {
  renderBootError(event.error || event.message);
});

window.addEventListener("unhandledrejection", (event) => {
  renderBootError(event.reason);
});

try {
  const root = document.getElementById("root");
  if (!root) {
    throw new Error("找不到前端挂载节点");
  }

  ReactDOM.createRoot(root).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>
  );
} catch (error) {
  renderBootError(error);
}
