import React from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Badge, type IconName } from "../ui";

type TabIconProps = { name: IconName; focusedName: IconName; focused: boolean; color: string; size: number; badge?: number };

/** Icon + optional unread badge shown in the bottom tab bar. */
export function TabIcon({ name, focusedName, focused, color, size, badge = 0 }: TabIconProps) {
  return (
    <View>
      <Ionicons color={color} name={focused ? focusedName : name} size={size} />
      {badge > 0 ? (
        <View style={styles.badge}>
          <Badge count={badge} />
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  badge: { position: "absolute", top: -6, right: -12 }
});
