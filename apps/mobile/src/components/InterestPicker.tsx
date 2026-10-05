import { StyleSheet, View } from "react-native";
import { useInterests } from "../hooks/queries";
import { Chip, Loader, Text } from "../shared/ui";
import { spacing } from "../theme";

export const MAX_INTERESTS = 10;

export function InterestPicker({ selected, onChange }: { selected: string[]; onChange: (slugs: string[]) => void }) {
  const { data, isLoading, isError } = useInterests();
  if (isLoading) return <Loader />;
  if (isError || !data) return <Text tone="danger">Impossible de charger les centres d'intérêt.</Text>;

  const toggle = (slug: string) => {
    if (selected.includes(slug)) onChange(selected.filter((s) => s !== slug));
    else if (selected.length < MAX_INTERESTS) onChange([...selected, slug]);
  };

  return (
    <View>
      <View style={styles.wrap}>
        {data.map((interest) => (
          <Chip
            key={interest.slug}
            label={interest.label}
            selected={selected.includes(interest.slug)}
            disabled={!selected.includes(interest.slug) && selected.length >= MAX_INTERESTS}
            onPress={() => toggle(interest.slug)}
          />
        ))}
      </View>
      <Text variant="caption" tone="muted" style={styles.count}>
        {selected.length} / {MAX_INTERESTS} sélectionnés
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({ wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }, count: { marginTop: spacing.sm } });
