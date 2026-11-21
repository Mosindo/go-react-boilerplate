import { Platform } from "react-native";

// React Native Web keeps a fading modal mounted until its CSS animation ends,
// which never happens if the app navigates away in the same tick. Modals are
// therefore not animated on web.
export const modalAnimation = (native: "fade" | "slide"): "fade" | "slide" | "none" => (Platform.OS === "web" ? "none" : native);
