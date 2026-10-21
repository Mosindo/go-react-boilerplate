import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useUnreadCounts } from "../hooks/queries";
import type { RootStackParamList } from "../navigation/types";
import { Badge } from "../shared/ui";
import { MIN_TOUCH, useTheme } from "../theme";

export function BellButton() {
  const theme = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { notifications } = useUnreadCounts();
  return (
    <Pressable
      testID="open-notifications"
      accessibilityRole="button"
      accessibilityLabel={notifications > 0 ? `Notifications, ${notifications} non lue(s)` : "Notifications"}
      onPress={() => navigation.navigate("Notifications")}
      style={styles.button}
    >
      <Ionicons name={notifications > 0 ? "notifications" : "notifications-outline"} size={26} color={theme.text} />
      {notifications > 0 ? (
        <View style={styles.badge}>
          <Badge count={notifications} testID="notifications-badge" />
        </View>
      ) : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  button: { width: MIN_TOUCH, height: MIN_TOUCH, alignItems: "center", justifyContent: "center" },
  badge: { position: "absolute", top: 2, right: 0 }
});
