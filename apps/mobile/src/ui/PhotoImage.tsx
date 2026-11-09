import React, { useEffect, useState } from "react";
import { Image, Platform, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { API_URL } from "../config";
import { getValidAccessToken } from "../api/client";
import { useTheme } from "../theme/theme";
import { Text } from "./Text";

type Props = {
  url?: string;
  name?: string;
  style?: StyleProp<ViewStyle>;
  circle?: boolean;
};

/**
 * Photos are private: they are fetched with the bearer token. The token is resolved
 * (and refreshed) before rendering so the image request is authorised.
 */
export function PhotoImage({ url, name, style, circle }: Props) {
  const { colors } = useTheme();
  const [token, setToken] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  const [webUri, setWebUri] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    getValidAccessToken().then((t) => {
      if (!cancelled) {
        setToken(t);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [url]);

  // The web <img> element cannot send an Authorization header, so web builds fetch the bytes first.
  useEffect(() => {
    if (Platform.OS !== "web" || !url || !token) {
      return;
    }
    let objectUrl: string | null = null;
    let cancelled = false;
    fetch(`${API_URL}${url}`, { headers: { Authorization: `Bearer ${token}` } })
      .then((res) => (res.ok ? res.blob() : Promise.reject(new Error("photo"))))
      .then((blob) => {
        objectUrl = URL.createObjectURL(blob);
        if (!cancelled) setWebUri(objectUrl);
      })
      .catch(() => {
        if (!cancelled) setFailed(true);
      });
    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [url, token]);

  const isWeb = Platform.OS === "web";
  const source = isWeb
    ? webUri
      ? { uri: webUri }
      : null
    : url && token
      ? { uri: `${API_URL}${url}`, headers: { Authorization: `Bearer ${token}` } }
      : null;
  const showImage = !!source && !failed;
  return (
    <View style={[styles.box, { backgroundColor: colors.surfaceAlt }, circle ? styles.circle : null, style]}>
      {showImage ? (
        <Image
          accessibilityIgnoresInvertColors
          source={source}
          style={StyleSheet.absoluteFill}
          resizeMode="cover"
          onError={() => setFailed(true)}
        />
      ) : (
        <Text variant="heading" muted>
          {name ? name.charAt(0).toUpperCase() : "?"}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  box: { overflow: "hidden", alignItems: "center", justifyContent: "center" },
  circle: { borderRadius: 999 }
});
