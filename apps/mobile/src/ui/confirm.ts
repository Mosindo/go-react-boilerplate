import { Alert, Platform } from "react-native";

type ConfirmOptions = { title: string; message?: string; confirmLabel?: string; destructive?: boolean };

/** Cross-platform confirmation dialog (Alert.alert is a no-op on web). */
export function confirm({ title, message, confirmLabel = "Confirmer", destructive }: ConfirmOptions): Promise<boolean> {
  if (Platform.OS === "web") {
    const text = message ? `${title}\n\n${message}` : title;
    return Promise.resolve(typeof window !== "undefined" ? window.confirm(text) : false);
  }
  return new Promise((resolve) => {
    Alert.alert(title, message, [
      { text: "Annuler", style: "cancel", onPress: () => resolve(false) },
      { text: confirmLabel, style: destructive ? "destructive" : "default", onPress: () => resolve(true) }
    ]);
  });
}
