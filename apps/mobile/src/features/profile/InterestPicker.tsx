import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterests } from "../../state/queries";
import { Chip, LoadingState, Text } from "../../ui";

type Props = { selected: string[]; onChange: (slugs: string[]) => void; max?: number };

export function InterestPicker({ selected, onChange, max = 10 }: Props) {
  const { data, isLoading } = useInterests();
  if (isLoading || !data) {
    return <LoadingState />;
  }
  const toggle = (slug: string) => {
    if (selected.includes(slug)) {
      onChange(selected.filter((s) => s !== slug));
    } else if (selected.length < max) {
      onChange([...selected, slug]);
    }
  };
  return (
    <View style={styles.wrap}>
      <Text variant="caption" tone="muted">
        {selected.length}/{max} sélectionnés
      </Text>
      <View style={styles.chips}>
        {data.map((interest) => (
          <Chip
            key={interest.slug}
            label={interest.label}
            selected={selected.includes(interest.slug)}
            onPress={() => toggle(interest.slug)}
            testID={`interest-${interest.slug}`}
          />
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 10 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
