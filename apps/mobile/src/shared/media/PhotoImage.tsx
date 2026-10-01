import React, { useState } from "react";
import { Image, StyleSheet, View, type ImageStyle, type StyleProp } from "react-native";
import { API_BASE_URL } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { Text } from "../ui/Text";
import { useTheme } from "../ui/theme";

type PhotoImageProps = {
  /** API-relative path such as /photos/:id/file. */
  path?: string | null;
  fallbackLabel?: string;
  style?: StyleProp<ImageStyle>;
  accessibilityLabel?: string;
};

/**
 * Photos are private: they are fetched with the bearer token instead of a
 * public URL. RN's Image forwards `headers` natively and caches by URL.
 */
export function PhotoImage({ path, fallbackLabel, style, accessibilityLabel }: PhotoImageProps) {
  const { accessToken } = useAuth();
  const { colors } = useTheme();
  const [failed, setFailed] = useState(false);

  if (!path || !accessToken || failed) {
    return (
      <View style={[styles.fallback, { backgroundColor: colors.surfaceSubtle }, style as object]}>
        <Text variant="heading" tone="muted">
          {(fallbackLabel ?? "?").slice(0, 1).toUpperCase()}
        </Text>
      </View>
    );
  }

  return (
    <Image
      accessibilityIgnoresInvertColors
      accessibilityLabel={accessibilityLabel}
      onError={() => setFailed(true)}
      resizeMode="cover"
      source={{ uri: `${API_BASE_URL}${path}`, headers: { Authorization: `Bearer ${accessToken}` }, cache: "force-cache" }}
      style={[{ backgroundColor: colors.surfaceSubtle }, style]}
    />
  );
}

const styles = StyleSheet.create({ fallback: { alignItems: "center", justifyContent: "center" } });
