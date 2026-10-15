import { Alert } from "react-native";

type ConfirmOptions = {
  title: string;
  message?: string;
  confirmLabel: string;
  destructive?: boolean;
  onConfirm: () => void;
};

/** Native two-button confirmation for irreversible actions. */
export function confirm({ confirmLabel, destructive = false, message, onConfirm, title }: ConfirmOptions): void {
  Alert.alert(title, message, [
    { text: "Annuler", style: "cancel" },
    { text: confirmLabel, style: destructive ? "destructive" : "default", onPress: onConfirm }
  ]);
}
