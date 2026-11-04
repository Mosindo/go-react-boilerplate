import { Alert } from "react-native";

/** Native confirmation dialog. `destructive` styles the confirm button in red on iOS. */
export function confirm(title: string, message: string, confirmLabel: string, onConfirm: () => void, destructive = true): void {
  Alert.alert(title, message, [
    { text: "Annuler", style: "cancel" },
    { text: confirmLabel, style: destructive ? "destructive" : "default", onPress: onConfirm },
  ]);
}
