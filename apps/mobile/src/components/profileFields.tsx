import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Interest, LookingFor, Preferences } from "../api/types";
import { readDeviceLocation } from "../lib/location";
import { CITIES, searchCities } from "../lib/cities";
import { DISTANCE_CHOICES, GENDERS, LOOKING_FOR } from "../lib/labels";
import { BIO_MAX, MAX_AGE, MIN_AGE, type DateParts } from "../lib/validation";
import { Button, FormField, Input, Notice, Text, colors, radii, spacing } from "../shared/ui";
import { Chip, ChipRow } from "./Chip";
import { DateInput } from "./DateInput";
import { Stepper } from "./Stepper";

// ---------------------------------------------------------------- identity

export type IdentityState = { firstName: string; birthDate: DateParts; gender: Gender | null };

type IdentityErrors = { firstName?: string | null; birthDate?: string | null; gender?: string | null };

export function IdentityFields({
  value,
  onChange,
  errors = {},
  birthDateLocked = false
}: {
  value: IdentityState;
  onChange: (next: IdentityState) => void;
  errors?: IdentityErrors;
  birthDateLocked?: boolean;
}) {
  return (
    <View style={styles.stack}>
      <Input
        autoCapitalize="words"
        autoComplete="given-name"
        error={errors.firstName}
        label="Prénom"
        maxLength={40}
        onChangeText={(firstName) => onChange({ ...value, firstName })}
        placeholder="Votre prénom"
        testID="input-first-name"
        textContentType="givenName"
        value={value.firstName}
      />
      <DateInput
        disabled={birthDateLocked}
        error={errors.birthDate}
        onChange={(birthDate) => onChange({ ...value, birthDate })}
        value={value.birthDate}
      />
      {birthDateLocked ? (
        <Text tone="muted" variant="caption">
          La date de naissance ne peut pas être modifiée.
        </Text>
      ) : null}
      <FormField error={errors.gender} label="Je suis">
        <ChipRow>
          {GENDERS.map((item) => (
            <Chip
              key={item.value}
              label={item.label}
              onPress={() => onChange({ ...value, gender: item.value })}
              selected={value.gender === item.value}
              testID={`gender-${item.value}`}
            />
          ))}
        </ChipRow>
      </FormField>
    </View>
  );
}

// ---------------------------------------------------------------- preferences

export function PreferencesFields({ value, onChange }: { value: Preferences; onChange: (next: Preferences) => void }) {
  const toggle = (gender: Gender) => {
    const has = value.interestedIn.includes(gender);
    const next = has ? value.interestedIn.filter((g) => g !== gender) : [...value.interestedIn, gender];
    onChange({ ...value, interestedIn: next });
  };

  return (
    <View style={styles.stack}>
      <FormField helperText="Vous pouvez en choisir plusieurs." label="Je souhaite rencontrer">
        <ChipRow>
          {GENDERS.map((item) => (
            <Chip
              key={item.value}
              label={item.plural}
              onPress={() => toggle(item.value)}
              selected={value.interestedIn.includes(item.value)}
              testID={`seek-${item.value}`}
            />
          ))}
        </ChipRow>
      </FormField>

      <Stepper
        format={(v) => `${v} ans`}
        label="Âge minimum"
        max={value.maxAge}
        min={MIN_AGE}
        onChange={(minAge) => onChange({ ...value, minAge, maxAge: Math.max(value.maxAge, minAge) })}
        testID="stepper-min-age"
        value={value.minAge}
      />
      <Stepper
        format={(v) => `${v} ans`}
        label="Âge maximum"
        max={MAX_AGE}
        min={value.minAge}
        onChange={(maxAge) => onChange({ ...value, maxAge })}
        testID="stepper-max-age"
        value={value.maxAge}
      />

      <FormField helperText="Seuls les profils dans ce rayon (et qui acceptent le vôtre) vous seront proposés." label="Distance maximale">
        <ChipRow>
          {DISTANCE_CHOICES.map((km) => (
            <Chip
              key={km}
              label={km === 500 ? "500 km +" : `${km} km`}
              onPress={() => onChange({ ...value, maxDistanceKm: km })}
              selected={value.maxDistanceKm === km}
              testID={`distance-${km}`}
            />
          ))}
        </ChipRow>
      </FormField>
    </View>
  );
}

// ---------------------------------------------------------------- location

export type LocationState = { latitude: number; longitude: number; city: string } | null;

export function LocationPicker({ value, onChange }: { value: LocationState; onChange: (next: LocationState) => void }) {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [showCities, setShowCities] = useState(false);

  const shareDeviceLocation = async () => {
    setBusy(true);
    setMessage(null);
    const result = await readDeviceLocation();
    setBusy(false);
    if (result.status === "ok") {
      onChange(result.location);
      setShowCities(false);
    } else if (result.status === "denied") {
      setMessage("Localisation refusée. Choisissez votre ville dans la liste.");
      setShowCities(true);
    } else {
      setMessage("Position indisponible. Choisissez votre ville dans la liste.");
      setShowCities(true);
    }
  };

  const cities = showCities ? (query ? searchCities(query) : CITIES).slice(0, 8) : [];

  return (
    <View style={styles.stack}>
      <Text tone="muted">
        Votre position exacte n'est jamais montrée : les autres voient seulement une distance approximative. Nous ne gardons qu'une position
        arrondie à environ 1 km.
      </Text>
      {value ? (
        <Notice
          description="Position enregistrée (arrondie)."
          title={value.city ? `📍 ${value.city}` : "📍 Position enregistrée"}
          tone="success"
        />
      ) : null}
      <Button
        label={value ? "Mettre à jour ma position" : "Utiliser ma position"}
        loading={busy}
        onPress={() => void shareDeviceLocation()}
        testID="use-location"
        variant={value ? "outline" : "primary"}
      />
      <Button label="Choisir une ville" onPress={() => setShowCities((current) => !current)} testID="choose-city" variant="ghost" />
      {message ? <Notice title={message} tone="warning" /> : null}
      {showCities ? (
        <View style={styles.stack}>
          <Input autoCapitalize="words" onChangeText={setQuery} placeholder="Rechercher une ville" testID="city-search" value={query} />
          <ChipRow>
            {cities.map((city) => (
              <Chip
                key={city.name}
                label={city.name}
                onPress={() => {
                  onChange({ latitude: city.latitude, longitude: city.longitude, city: city.name });
                  setShowCities(false);
                  setMessage(null);
                }}
                selected={value?.city === city.name}
                testID={`city-${city.name}`}
              />
            ))}
          </ChipRow>
          {cities.length === 0 ? <Text tone="muted">Aucune ville ne correspond.</Text> : null}
        </View>
      ) : null}
    </View>
  );
}

// ---------------------------------------------------------------- about

export type AboutState = { bio: string; occupation: string; lookingFor: LookingFor; interests: string[] };

export const MAX_INTERESTS = 10;

export function AboutFields({
  value,
  onChange,
  catalogue
}: {
  value: AboutState;
  onChange: (next: AboutState) => void;
  catalogue: Interest[];
}) {
  const toggle = (slug: string) => {
    if (value.interests.includes(slug)) {
      onChange({ ...value, interests: value.interests.filter((s) => s !== slug) });
    } else if (value.interests.length < MAX_INTERESTS) {
      onChange({ ...value, interests: [...value.interests, slug] });
    }
  };

  return (
    <View style={styles.stack}>
      <Input
        helperText={`${value.bio.length}/${BIO_MAX}`}
        label="À propos de moi"
        maxLength={BIO_MAX}
        multiline
        onChangeText={(bio) => onChange({ ...value, bio })}
        placeholder="Quelques mots pour vous présenter…"
        testID="input-bio"
        value={value.bio}
      />
      <Input
        label="Profession"
        maxLength={80}
        onChangeText={(occupation) => onChange({ ...value, occupation })}
        placeholder="Facultatif"
        testID="input-occupation"
        value={value.occupation}
      />
      <FormField label="Je cherche">
        <ChipRow>
          {LOOKING_FOR.map((item) => (
            <Chip
              key={item.value}
              label={item.label}
              onPress={() => onChange({ ...value, lookingFor: item.value })}
              selected={value.lookingFor === item.value}
            />
          ))}
        </ChipRow>
      </FormField>
      <FormField helperText={`${value.interests.length}/${MAX_INTERESTS} sélectionnés`} label="Centres d'intérêt">
        <ChipRow>
          {catalogue.map((interest) => (
            <Chip
              disabled={!value.interests.includes(interest.slug) && value.interests.length >= MAX_INTERESTS}
              key={interest.slug}
              label={interest.label}
              onPress={() => toggle(interest.slug)}
              selected={value.interests.includes(interest.slug)}
              testID={`interest-${interest.slug}`}
            />
          ))}
        </ChipRow>
      </FormField>
    </View>
  );
}

export function SectionCard({ children }: { children: React.ReactNode }) {
  return <View style={styles.card}>{children}</View>;
}

const styles = StyleSheet.create({
  stack: { gap: spacing.lg },
  card: {
    backgroundColor: colors.backgroundElevated,
    borderRadius: radii.lg,
    borderWidth: 1,
    borderColor: colors.border,
    padding: spacing.lg,
    gap: spacing.lg
  }
});
