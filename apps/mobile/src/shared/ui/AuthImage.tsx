import React, { useMemo, useState, useSyncExternalStore } from "react";
import { Image, StyleSheet, View, type ImageStyle, type StyleProp } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { API_BASE_URL, http, tokenStore } from "../../api";
import { useTheme } from "../../theme";

type Props = {
  /** API-relative path, e.g. Photo.url or Photo.thumbUrl. */
  path: string | null | undefined;
  style?: StyleProp<ImageStyle>;
  resizeMode?: "cover" | "contain";
  accessibilityLabel?: string;
};

const subscribeToken = (listener: () => void) => tokenStore.subscribe(listener);
const readToken = () => tokenStore.getAccessToken();
const FORCED_REFRESH_COOLDOWN_MS = 10_000;
let lastForcedRefresh = 0;

/** Images are served behind the Bearer header, never through a token in the URL. */
export const AuthImage = React.memo(function AuthImage({
  path,
  style,
  resizeMode = "cover",
  accessibilityLabel
}: Props) {
  const { colors } = useTheme();
  const token = useSyncExternalStore(subscribeToken, readToken);
  const [failedToken, setFailedToken] = useState<string | null>(null);

  const source = useMemo(
    () => (path && token ? { uri: `${API_BASE_URL}${path}`, headers: { Authorization: `Bearer ${token}` } } : null),
    [path, token]
  );

  const handleError = () => {
    setFailedToken(token);
    const nowMs = Date.now();
    if (nowMs - lastForcedRefresh > FORCED_REFRESH_COOLDOWN_MS) {
      lastForcedRefresh = nowMs;
      // A rejected image is most often an expired access token: renew it, the new token re-renders this image.
      http.getFreshAccessToken(true).catch(() => undefined);
    }
  };

  if (!source || failedToken === token) {
    return (
      <View style={[styles.placeholder, { backgroundColor: colors.surfaceAlt }, style]}>
        <Ionicons color={colors.textMuted} name="person" size={32} />
      </View>
    );
  }

  return (
    <Image
      accessibilityIgnoresInvertColors
      accessibilityLabel={accessibilityLabel}
      accessible={Boolean(accessibilityLabel)}
      onError={handleError}
      resizeMode={resizeMode}
      source={source}
      style={[{ backgroundColor: colors.surfaceAlt }, style]}
    />
  );
});

const styles = StyleSheet.create({
  placeholder: { alignItems: "center", justifyContent: "center", overflow: "hidden" }
});
