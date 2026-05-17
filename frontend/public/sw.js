self.addEventListener("push", (event) => {
  const data = event.data?.json() || {};
  const title = data.title || "Confirmaciones";
  const options = {
    body: data.body || "Hay casos pendientes de revisión",
    icon: "/icon-192.png",
    badge: "/icon-192.png",
    data: { url: data.url || "/review" },
    vibrate: [200, 100, 200],
    tag: "confirmation-review",
    requireInteraction: true,
  };

  event.waitUntil(
    self.registration.showNotification(title, options)
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const url = event.notification.data?.url || "/review";
  event.waitUntil(
    clients.matchAll({ type: "window" }).then((clientList) => {
      for (const client of clientList) {
        if (client.url.includes(url) && "focus" in client) {
          return client.focus();
        }
      }
      if (clients.openWindow) return clients.openWindow(url);
    })
  );
});
