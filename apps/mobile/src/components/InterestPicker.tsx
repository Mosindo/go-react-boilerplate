import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterests } from "../hooks/queries";
import { spacing } from "../theme";
import { Chip, LoadingState, Text } from "../ui";

export const MAX_INTERESTS = 10;

type InterestPickerProps = {
  value: number[];
  onChange: (ids: number[]) => void;
};

export function InterestPicker({ value, onChange }: InterestPickerProps) {
  const interests = useInterests();
  if (interests.isLoading) {
    return <LoadingState />;
  }
  const toggle = (id: number) => {
    if (value.includes(id)) {
      onChange(value.filter((v) => v !== id));
    } else if (value.length < MAX_INTERESTS) {
      onChange([...value, id]);
    }
  };
  return (
    <View style={styles.wrap}>
      <Text variant="caption" color="textSubtle">
        {value.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      <View style={styles.chips}>
        {(interests.data ?? []).map((i) => (
          <Chip key={i.id} label={i.label} selected={value.includes(i.id)} onPress={() => toggle(i.id)} testID={`interest-${i.slug}`} />
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    gap: spacing.xs
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  }
});
