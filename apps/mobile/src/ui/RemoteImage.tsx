import React, { useEffect, useState } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image } from "expo-image";
import { API_BASE_URL, currentAccessToken } from "../api/client";
import { photoUri } from "../lib/url";
import { useTheme } from "../theme";

/** Photos are private: the bearer token must travel with each image request. */
export function RemoteImage({
  url,
  style,
  accessibilityLabel
}: {
  url: string | null;
  style?: StyleProp<ViewStyle>;
  accessibilityLabel?: string;
}) {
  const { colors } = useTheme();
  const [token, setToken] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    void currentAccessToken().then((t) => {
      if (active) setToken(t);
    });
    return () => {
      active = false;
    };
  }, [url]);

  if (!url || !token) {
    return <View style={[styles.placeholder, { backgroundColor: colors.surfaceAlt }, style]} />;
  }
  const uri = photoUri(API_BASE_URL, url);
  return (
    <Image
      source={{ uri, headers: { Authorization: `Bearer ${token}` }, cacheKey: uri }}
      style={style as never}
      contentFit="cover"
      transition={150}
      cachePolicy="disk"
      accessibilityLabel={accessibilityLabel}
      accessible={Boolean(accessibilityLabel)}
    />
  );
}

const styles = StyleSheet.create({ placeholder: { overflow: "hidden" } });
