import { Alert, Platform } from "react-native";

/** Cross-platform confirmation (Alert is a no-op on web). */
export function confirmAction(title: string, message: string, actionLabel: string, onConfirm: () => void): void {
  if (Platform.OS === "web") {
    if (globalThis.confirm?.(`${title}\n${message}`)) {
      onConfirm();
    }
    return;
  }
  Alert.alert(title, message, [
    { text: "Annuler", style: "cancel" },
    { text: actionLabel, style: "destructive", onPress: onConfirm }
  ]);
}

export function notify(title: string, message?: string): void {
  if (Platform.OS === "web") {
    globalThis.alert?.(message ? `${title}\n${message}` : title);
    return;
  }
  Alert.alert(title, message);
}
