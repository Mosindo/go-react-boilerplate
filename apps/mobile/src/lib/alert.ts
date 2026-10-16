import { Alert, Platform, type AlertButton } from "react-native";

/**
 * Alert.alert on phones; window.confirm/alert on the web build, where React Native's
 * Alert is a no-op (the web target is used for development and browser-driven tests).
 */
export function showAlert(title: string, message?: string, buttons?: AlertButton[]): void {
  if (Platform.OS !== "web") {
    Alert.alert(title, message, buttons);
    return;
  }
  const text = message ? `${title}\n\n${message}` : title;
  const cancel = buttons?.find((button) => button.style === "cancel");
  const confirm = buttons?.find((button) => button.style !== "cancel");
  if (buttons && buttons.length > 1) {
    if (window.confirm(text)) {
      confirm?.onPress?.();
    } else {
      cancel?.onPress?.();
    }
    return;
  }
  window.alert(text);
  buttons?.[0]?.onPress?.();
}
