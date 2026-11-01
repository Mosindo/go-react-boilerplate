import { View } from "react-native";
import { useInterests } from "../state/queries";
import { Chip, Loading, Text } from "./ui";

export const MAX_INTERESTS = 10;

export function InterestPicker({ selected, onChange }: { selected: string[]; onChange: (slugs: string[]) => void }) {
  const interests = useInterests();
  if (interests.isLoading) {
    return <Loading />;
  }
  const toggle = (slug: string) => {
    if (selected.includes(slug)) {
      onChange(selected.filter((s) => s !== slug));
    } else if (selected.length < MAX_INTERESTS) {
      onChange([...selected, slug]);
    }
  };
  return (
    <View>
      <View style={{ flexDirection: "row", flexWrap: "wrap" }}>
        {(interests.data ?? []).map((interest) => (
          <Chip
            key={interest.slug}
            label={interest.label}
            onPress={() => toggle(interest.slug)}
            selected={selected.includes(interest.slug)}
          />
        ))}
      </View>
      <Text muted variant="caption">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
    </View>
  );
}
