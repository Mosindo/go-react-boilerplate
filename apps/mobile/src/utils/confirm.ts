import { Alert, Platform } from "react-native";

/** Cross-platform confirmation dialog (react-native-web's Alert is a no-op). */
export function confirmAction(title: string, message: string, confirmLabel: string, destructive = true): Promise<boolean> {
  if (Platform.OS === "web") {
    return Promise.resolve(typeof window !== "undefined" ? window.confirm(`${title}\n\n${message}`) : false);
  }
  return new Promise((resolve) => {
    Alert.alert(title, message, [
      { text: "Annuler", style: "cancel", onPress: () => resolve(false) },
      { text: confirmLabel, style: destructive ? "destructive" : "default", onPress: () => resolve(true) }
    ]);
  });
}
