import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image } from "expo-image";
import { photoSource } from "../api/client";
import { colors } from "../shared/ui";
import { useAccessToken } from "../store/tokenStore";

type RemotePhotoProps = {
  url?: string;
  style?: StyleProp<ViewStyle>;
  accessibilityLabel?: string;
};

/** Photos are private: they are fetched with the bearer token and cached on disk per URL. */
export function RemotePhoto({ url, style, accessibilityLabel }: RemotePhotoProps) {
  const token = useAccessToken();
  if (!url) {
    return <View style={[styles.placeholder, style]} />;
  }
  return (
    <Image
      accessibilityLabel={accessibilityLabel}
      cachePolicy="disk"
      contentFit="cover"
      source={photoSource(url, token)}
      style={[styles.image, style as object]}
      transition={150}
    />
  );
}

const styles = StyleSheet.create({
  image: { backgroundColor: colors.surfaceSubtle },
  placeholder: { backgroundColor: colors.surfaceSubtle }
});
