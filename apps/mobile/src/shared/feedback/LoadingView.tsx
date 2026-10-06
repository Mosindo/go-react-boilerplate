import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Loader } from "../ui";

type LoadingViewProps = {
  label?: string;
  fullScreen?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function LoadingView({ label = "Chargement…", fullScreen = false, style, testID }: LoadingViewProps) {
  return (
    <View
      style={[fullScreen ? { flex: 1, alignItems: "center", justifyContent: "center" } : { paddingVertical: 48 }, style]}
      testID={testID}
    >
      <Loader label={label} />
    </View>
  );
}
