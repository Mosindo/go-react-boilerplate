import { Pressable, Switch, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { GENDER_LABELS } from "../api";
import type { Gender, Settings } from "../api/types";
import { MAX_AGE, MIN_AGE } from "../lib/date";
import { spacing, useColors } from "../theme";
import { Chip, Text } from "./ui";

const GENDERS: Gender[] = ["woman", "man", "nonbinary"];
const DISTANCES = [10, 25, 50, 100, 200, 500];

function Stepper({
  label,
  value,
  onChange,
  min,
  max,
}: {
  label: string;
  value: number;
  onChange: (value: number) => void;
  min: number;
  max: number;
}) {
  const colors = useColors();
  const button = (icon: "remove" | "add", delta: number, disabled: boolean) => (
    <Pressable
      accessibilityLabel={`${delta > 0 ? "Augmenter" : "Diminuer"} ${label}`}
      accessibilityRole="button"
      disabled={disabled}
      hitSlop={8}
      onPress={() => onChange(value + delta)}
      style={{
        width: 36,
        height: 36,
        borderRadius: 18,
        borderWidth: 1,
        borderColor: colors.border,
        alignItems: "center",
        justifyContent: "center",
        opacity: disabled ? 0.4 : 1,
      }}
    >
      <Ionicons color={colors.text} name={icon} size={18} />
    </Pressable>
  );
  return (
    <View
      style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between", marginBottom: spacing.sm }}
    >
      <Text>{label}</Text>
      <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md }}>
        {button("remove", -1, value <= min)}
        <Text style={{ minWidth: 32, textAlign: "center" }} variant="label">
          {value}
        </Text>
        {button("add", 1, value >= max)}
      </View>
    </View>
  );
}

export function PreferencesForm({
  value,
  onChange,
  showPrivacy = true,
}: {
  value: Settings;
  onChange: (next: Settings) => void;
  showPrivacy?: boolean;
}) {
  const colors = useColors();
  const toggleGender = (gender: Gender) => {
    const has = value.interestedIn.includes(gender);
    const next = has ? value.interestedIn.filter((g) => g !== gender) : [...value.interestedIn, gender];
    if (next.length > 0) {
      onChange({ ...value, interestedIn: next });
    }
  };

  return (
    <View>
      <Text style={{ marginBottom: spacing.sm }} variant="heading">
        Je souhaite rencontrer
      </Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", marginBottom: spacing.lg }}>
        {GENDERS.map((g) => (
          <Chip
            key={g}
            label={GENDER_LABELS[g]}
            onPress={() => toggleGender(g)}
            selected={value.interestedIn.includes(g)}
          />
        ))}
      </View>

      <Text style={{ marginBottom: spacing.sm }} variant="heading">
        Tranche d'âge
      </Text>
      <Stepper
        label="Âge minimum"
        max={value.maxAge}
        min={MIN_AGE}
        onChange={(minAge) => onChange({ ...value, minAge })}
        value={value.minAge}
      />
      <Stepper
        label="Âge maximum"
        max={MAX_AGE}
        min={value.minAge}
        onChange={(maxAge) => onChange({ ...value, maxAge })}
        value={value.maxAge}
      />

      <Text style={{ marginBottom: spacing.sm, marginTop: spacing.lg }} variant="heading">
        Distance maximale
      </Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap" }}>
        {DISTANCES.map((km) => (
          <Chip
            key={km}
            label={`${km} km`}
            onPress={() => onChange({ ...value, maxDistanceKm: km })}
            selected={value.maxDistanceKm === km}
          />
        ))}
      </View>

      {showPrivacy ? (
        <View style={{ marginTop: spacing.lg }}>
          <Text style={{ marginBottom: spacing.sm }} variant="heading">
            Confidentialité
          </Text>
          <View style={{ flexDirection: "row", alignItems: "center", marginBottom: spacing.md }}>
            <View style={{ flex: 1, paddingRight: spacing.md }}>
              <Text variant="label">Profil visible</Text>
              <Text muted variant="caption">
                Désactivé, vous n'apparaissez plus dans la découverte (vos matchs restent actifs).
              </Text>
            </View>
            <Switch
              accessibilityLabel="Profil visible"
              onValueChange={(discoverable) => onChange({ ...value, discoverable })}
              thumbColor={colors.surface}
              trackColor={{ true: colors.primary, false: colors.border }}
              value={value.discoverable}
            />
          </View>
          <View style={{ flexDirection: "row", alignItems: "center" }}>
            <View style={{ flex: 1, paddingRight: spacing.md }}>
              <Text variant="label">Afficher ma distance</Text>
              <Text muted variant="caption">
                Une distance approximative est montrée aux autres. Jamais votre position.
              </Text>
            </View>
            <Switch
              accessibilityLabel="Afficher ma distance"
              onValueChange={(showDistance) => onChange({ ...value, showDistance })}
              thumbColor={colors.surface}
              trackColor={{ true: colors.primary, false: colors.border }}
              value={value.showDistance}
            />
          </View>
        </View>
      ) : null}
    </View>
  );
}
