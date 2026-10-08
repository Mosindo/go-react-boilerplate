import { Alert, Platform } from "react-native";

type ConfirmOptions = { title: string; message?: string; confirmLabel?: string; destructive?: boolean };

/** Cross-platform yes/no dialog (Alert.alert is a no-op on web). */
export function confirm({ title, message, confirmLabel = "Confirmer", destructive = false }: ConfirmOptions): Promise<boolean> {
  if (Platform.OS === "web") {
    return Promise.resolve(typeof window !== "undefined" && window.confirm(message ? `${title}\n\n${message}` : title));
  }
  return new Promise((resolve) => {
    Alert.alert(
      title,
      message,
      [
        { text: "Annuler", style: "cancel", onPress: () => resolve(false) },
        { text: confirmLabel, style: destructive ? "destructive" : "default", onPress: () => resolve(true) }
      ],
      { cancelable: true, onDismiss: () => resolve(false) }
    );
  });
}
