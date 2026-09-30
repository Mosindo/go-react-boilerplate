import { Platform } from "react-native";
import Constants from "expo-constants";
import * as Device from "expo-device";
import * as Notifications from "expo-notifications";
import { notificationsApi } from "../api/endpoints";

let registeredToken: string | null = null;
const handledResponses = new Set<string>();

// In the foreground, realtime updates and in-app badges already inform the
// member, so system banners are only shown when the app is in background.
if (Platform.OS !== "web") {
  Notifications.setNotificationHandler({
    handleNotification: async () => ({
      shouldShowBanner: false,
      shouldShowList: true,
      shouldPlaySound: false,
      shouldSetBadge: false
    })
  });
}

function projectId(): string | undefined {
  const fromExtra = (Constants.expoConfig?.extra as { eas?: { projectId?: string } } | undefined)?.eas?.projectId;
  return fromExtra ?? Constants.easConfig?.projectId ?? undefined;
}

/**
 * Asks for permission (once, after sign-in) and registers the Expo push token.
 * Silently does nothing on web, simulators, when permission is denied, or when
 * the app has no EAS project id (push requires an EAS project).
 */
export async function registerForPush(): Promise<void> {
  if (Platform.OS === "web" || !Device.isDevice) return;
  const id = projectId();
  if (!id) return;

  if (Platform.OS === "android") {
    await Notifications.setNotificationChannelAsync("default", {
      name: "Matchs et messages",
      importance: Notifications.AndroidImportance.HIGH
    });
  }

  const current = await Notifications.getPermissionsAsync();
  let granted = current.granted;
  if (!granted && current.canAskAgain) {
    granted = (await Notifications.requestPermissionsAsync()).granted;
  }
  if (!granted) return;

  const { data: token } = await Notifications.getExpoPushTokenAsync({ projectId: id });
  await notificationsApi.registerPushToken(token, Platform.OS === "ios" ? "ios" : "android");
  registeredToken = token;
}

/** Stops pushes to this device for the account being signed out. */
export async function unregisterPush(): Promise<void> {
  if (!registeredToken) return;
  const token = registeredToken;
  registeredToken = null;
  await notificationsApi.unregisterPushToken(token).catch(() => undefined);
}

/** Calls `onOpen` with the conversation id when a push notification is tapped. */
export function onPushOpened(onOpen: (conversationId: string) => void): () => void {
  if (Platform.OS === "web") return () => undefined;
  const open = (response: Notifications.NotificationResponse | null) => {
    if (!response) return;
    // getLastNotificationResponseAsync returns the same tap on every mount
    // (e.g. after signing in again): act on each tap only once.
    const id = response.notification.request.identifier;
    if (handledResponses.has(id)) return;
    handledResponses.add(id);
    const conversationId = response.notification.request.content.data?.conversationId;
    if (typeof conversationId === "string") onOpen(conversationId);
  };
  void Notifications.getLastNotificationResponseAsync().then(open);
  const subscription = Notifications.addNotificationResponseReceivedListener(open);
  return () => subscription.remove();
}
