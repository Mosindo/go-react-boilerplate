import React, { useState } from "react";
import { ActivityIndicator, Image, StyleSheet, View, type StyleProp, type ViewStyle, type ImageStyle } from "react-native";
import { API_BASE_URL } from "../api/config";
import { peekAccessToken } from "../store/tokenStore";
import { useTheme } from "../theme";
import { Text } from "./Text";

type Props = {
  url: string | null | undefined;
  style?: StyleProp<ViewStyle>;
  imageStyle?: StyleProp<ImageStyle>;
  resizeMode?: "cover" | "contain";
  label?: string;
  fallbackInitial?: string;
};

/** Photos are private: they are fetched with the Bearer token. */
export function AuthImage({ url, style, imageStyle, resizeMode = "cover", label, fallbackInitial }: Props) {
  const { colors } = useTheme();
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const token = peekAccessToken();

  const showFallback = !url || state === "error";
  return (
    <View style={[styles.box, { backgroundColor: colors.skeleton }, style]}>
      {url && state !== "error" ? (
        <Image
          accessibilityLabel={label}
          source={{ uri: `${API_BASE_URL}${url}`, headers: token ? { Authorization: `Bearer ${token}` } : undefined }}
          resizeMode={resizeMode}
          style={[StyleSheet.absoluteFill, imageStyle]}
          onLoad={() => setState("ready")}
          onError={() => setState("error")}
        />
      ) : null}
      {url && state === "loading" ? <ActivityIndicator color={colors.textMuted} /> : null}
      {showFallback ? (
        <Text variant="heading" tone="muted">
          {fallbackInitial ? fallbackInitial.slice(0, 1).toUpperCase() : "•"}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  box: { overflow: "hidden", alignItems: "center", justifyContent: "center" }
});
