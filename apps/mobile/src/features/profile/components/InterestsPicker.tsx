import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { MAX_INTERESTS } from "../../../config";
import { Button, Chip, Text } from "../../../shared/ui";
import { spacing } from "../../../theme";
import { useInterestsCatalog } from "../hooks";

type Props = { selected: string[]; onChange: (slugs: string[]) => void };

export function InterestsPicker({ selected, onChange }: Props) {
  const { data, isLoading, isError, refetch } = useInterestsCatalog();
  const atLimit = selected.length >= MAX_INTERESTS;

  const toggle = (slug: string) => {
    onChange(selected.includes(slug) ? selected.filter((item) => item !== slug) : [...selected, slug]);
  };

  return (
    <View style={styles.wrapper}>
      <Text color="textMuted" variant="caption">
        {`Centres d'intérêt (${selected.length}/${MAX_INTERESTS})`}
      </Text>
      {isLoading ? <ActivityIndicator /> : null}
      {isError ? (
        <Button label="Réessayer de charger les intérêts" onPress={() => void refetch()} variant="secondary" />
      ) : null}
      <View style={styles.chips}>
        {data?.map((interest) => {
          const isSelected = selected.includes(interest.slug);
          return (
            <Chip
              disabled={atLimit && !isSelected}
              key={interest.slug}
              label={interest.label}
              onPress={() => toggle(interest.slug)}
              selected={isSelected}
              testID={`interest-${interest.slug}`}
            />
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrapper: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
