import React, { useState } from "react";
import { Platform, Switch, View } from "react-native";
import * as Location from "expo-location";
import { useQuery } from "@tanstack/react-query";
import { AppText, Banner, Button, Chip, Row, TextField } from "./ui";
import { profileApi } from "../api/services";
import { spacing } from "../theme/theme";
import { validateBirthDate } from "../utils/validation";
import type { Gender, MyProfile, ProfileInput } from "../api/types";

export type Draft = {
  firstName: string;
  birthDate: string;
  gender: Gender | null;
  bio: string;
  city: string;
  latitude: number | null;
  longitude: number | null;
  interests: string[];
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
  showDistance: boolean;
  discoverable: boolean;
};

export const ANY_DISTANCE = 20000;
const GENDERS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Femme" },
  { value: "man", label: "Homme" },
  { value: "nonbinary", label: "Non-binaire" },
];

export function draftFrom(p: MyProfile | null | undefined): Draft {
  return {
    firstName: p?.firstName ?? "",
    birthDate: p?.birthDate ?? "",
    gender: p?.gender ?? null,
    bio: p?.bio ?? "",
    city: p?.city ?? "",
    latitude: null,
    longitude: null,
    interests: p?.interests.map((i) => i.slug) ?? [],
    interestedIn: p?.preferences.interestedIn ?? [],
    ageMin: p?.preferences.ageMin ?? 18,
    ageMax: p?.preferences.ageMax ?? 45,
    maxDistanceKm: p?.preferences.maxDistanceKm ?? 50,
    showDistance: p?.showDistance ?? true,
    discoverable: p?.discoverable ?? true,
  };
}

export type Section = "identity" | "about" | "preferences" | "privacy";

/** Returns a map of field -> message for the given sections (empty when valid). */
export function validateDraft(d: Draft, sections: Section[]): Record<string, string> {
  const errors: Record<string, string> = {};
  if (sections.includes("identity")) {
    if (!d.firstName.trim()) errors.firstName = "Votre prénom est requis.";
    const dateProblem = validateBirthDate(d.birthDate);
    if (dateProblem) errors.birthDate = dateProblem;
    if (!d.gender) errors.gender = "Choisissez une option.";
  }
  if (sections.includes("about") && d.bio.length > 500) errors.bio = "500 caractères maximum.";
  if (sections.includes("preferences")) {
    if (d.interestedIn.length === 0) errors.interestedIn = "Choisissez au moins une option.";
    if (d.ageMin > d.ageMax) errors.age = "La tranche d'âge est incohérente.";
  }
  return errors;
}

export function toInput(d: Draft): ProfileInput {
  if (!d.gender) throw new Error("gender is required");
  return {
    firstName: d.firstName.trim(),
    birthDate: d.birthDate.trim(),
    gender: d.gender,
    bio: d.bio.trim(),
    city: d.city.trim(),
    latitude: d.latitude,
    longitude: d.longitude,
    showDistance: d.showDistance,
    discoverable: d.discoverable,
    interests: d.interests,
    preferences: {
      interestedIn: d.interestedIn,
      ageMin: d.ageMin,
      ageMax: d.ageMax,
      maxDistanceKm: d.maxDistanceKm,
    },
  };
}

type SectionProps = {
  draft: Draft;
  set: (patch: Partial<Draft>) => void;
  errors: Record<string, string>;
  /** When the profile already has a location, editing without re-detecting keeps it. */
  hadLocation?: boolean;
};

export function formatBirthDateInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  const parts = [digits.slice(0, 4), digits.slice(4, 6), digits.slice(6, 8)].filter(Boolean);
  return parts.join("-");
}

export function IdentitySection({ draft, set, errors, hadLocation }: SectionProps) {
  const [locating, setLocating] = useState(false);
  const [locationNote, setLocationNote] = useState<string | null>(null);

  const locate = async () => {
    setLocating(true);
    setLocationNote(null);
    try {
      const perm = await Location.requestForegroundPermissionsAsync();
      if (perm.status !== "granted") {
        setLocationNote(
          "Position refusée. Vous pouvez saisir votre ville, mais la distance ne sera pas calculée.",
        );
        return;
      }
      const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      // The server rounds to ~1 km; we send only what is needed and never store the exact position.
      const patch: Partial<Draft> = { latitude: pos.coords.latitude, longitude: pos.coords.longitude };
      if (!draft.city.trim() && Platform.OS !== "web") {
        try {
          const [place] = await Location.reverseGeocodeAsync(pos.coords);
          if (place?.city) patch.city = place.city;
        } catch {
          // city can be typed manually
        }
      }
      set(patch);
      setLocationNote("Position approximative enregistrée (précision d'environ 1 km).");
    } catch {
      setLocationNote("Impossible de déterminer votre position pour le moment.");
    } finally {
      setLocating(false);
    }
  };

  return (
    <View style={{ gap: spacing.lg }}>
      <TextField
        testID="firstName"
        label="Prénom"
        value={draft.firstName}
        onChangeText={(v) => set({ firstName: v })}
        maxLength={40}
        error={errors.firstName}
        autoComplete="given-name"
      />
      <TextField
        testID="birthDate"
        label="Date de naissance"
        value={draft.birthDate}
        onChangeText={(v) => set({ birthDate: formatBirthDateInput(v) })}
        placeholder="AAAA-MM-JJ"
        keyboardType="number-pad"
        maxLength={10}
        error={errors.birthDate}
        hint="Votre âge est affiché, jamais votre date de naissance."
      />
      <View style={{ gap: spacing.sm }}>
        <AppText variant="small" style={{ fontWeight: "600" }}>
          Je suis
        </AppText>
        <Row style={{ flexWrap: "wrap" }}>
          {GENDERS.map((g) => (
            <Chip
              key={g.value}
              testID={`gender-${g.value}`}
              label={g.label}
              selected={draft.gender === g.value}
              onPress={() => set({ gender: g.value })}
            />
          ))}
        </Row>
        {errors.gender ? <AppText variant="error">{errors.gender}</AppText> : null}
      </View>
      <TextField
        testID="city"
        label="Ville"
        value={draft.city}
        onChangeText={(v) => set({ city: v })}
        maxLength={80}
        autoComplete="off"
      />
      <Button
        testID="locate"
        title={
          draft.latitude !== null || hadLocation
            ? "Mettre à jour ma position"
            : "Utiliser ma position approximative"
        }
        variant="secondary"
        loading={locating}
        onPress={locate}
      />
      {locationNote ? <AppText variant="muted">{locationNote}</AppText> : null}
    </View>
  );
}

export function AboutSection({ draft, set, errors }: SectionProps) {
  const catalog = useQuery({ queryKey: ["interests"], queryFn: profileApi.interests, staleTime: Infinity });
  const toggle = (slug: string) => {
    const has = draft.interests.includes(slug);
    if (!has && draft.interests.length >= 10) return;
    set({ interests: has ? draft.interests.filter((s) => s !== slug) : [...draft.interests, slug] });
  };
  return (
    <View style={{ gap: spacing.lg }}>
      <TextField
        testID="bio"
        label="À propos de moi"
        value={draft.bio}
        onChangeText={(v) => set({ bio: v })}
        multiline
        maxLength={500}
        error={errors.bio}
        hint={`${draft.bio.length}/500`}
      />
      <View style={{ gap: spacing.sm }}>
        <AppText variant="small" style={{ fontWeight: "600" }}>
          Centres d&apos;intérêt ({draft.interests.length}/10)
        </AppText>
        {catalog.isError ? <Banner message="Impossible de charger les centres d'intérêt." /> : null}
        <Row style={{ flexWrap: "wrap", gap: spacing.sm }}>
          {(catalog.data ?? []).map((i) => (
            <Chip
              key={i.slug}
              testID={`interest-${i.slug}`}
              label={i.label}
              selected={draft.interests.includes(i.slug)}
              onPress={() => toggle(i.slug)}
            />
          ))}
        </Row>
      </View>
    </View>
  );
}

function Stepper({
  label,
  value,
  min,
  max,
  onChange,
  testID,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (v: number) => void;
  testID: string;
}) {
  return (
    <Row style={{ justifyContent: "space-between" }}>
      <AppText>{label}</AppText>
      <Row>
        <Button
          title="−"
          variant="secondary"
          style={{ minWidth: 48, paddingHorizontal: 0 }}
          onPress={() => onChange(Math.max(min, value - 1))}
          testID={`${testID}-minus`}
        />
        <AppText style={{ minWidth: 32, textAlign: "center", fontWeight: "700" }} testID={`${testID}-value`}>
          {value}
        </AppText>
        <Button
          title="+"
          variant="secondary"
          style={{ minWidth: 48, paddingHorizontal: 0 }}
          onPress={() => onChange(Math.min(max, value + 1))}
          testID={`${testID}-plus`}
        />
      </Row>
    </Row>
  );
}

const DISTANCES: { value: number; label: string }[] = [
  { value: 10, label: "10 km" },
  { value: 25, label: "25 km" },
  { value: 50, label: "50 km" },
  { value: 100, label: "100 km" },
  { value: 300, label: "300 km" },
  { value: ANY_DISTANCE, label: "Partout" },
];

export function PreferencesSection({ draft, set, errors }: SectionProps) {
  const toggle = (g: Gender) =>
    set({
      interestedIn: draft.interestedIn.includes(g)
        ? draft.interestedIn.filter((x) => x !== g)
        : [...draft.interestedIn, g],
    });
  return (
    <View style={{ gap: spacing.lg }}>
      <View style={{ gap: spacing.sm }}>
        <AppText variant="small" style={{ fontWeight: "600" }}>
          Je souhaite rencontrer
        </AppText>
        <Row style={{ flexWrap: "wrap" }}>
          {GENDERS.map((g) => (
            <Chip
              key={g.value}
              testID={`interested-${g.value}`}
              label={g.label}
              selected={draft.interestedIn.includes(g.value)}
              onPress={() => toggle(g.value)}
            />
          ))}
        </Row>
        {errors.interestedIn ? <AppText variant="error">{errors.interestedIn}</AppText> : null}
      </View>
      <View style={{ gap: spacing.sm }}>
        <AppText variant="small" style={{ fontWeight: "600" }}>
          Tranche d&apos;âge
        </AppText>
        <Stepper
          label="Minimum"
          value={draft.ageMin}
          min={18}
          max={draft.ageMax}
          onChange={(v) => set({ ageMin: v })}
          testID="age-min"
        />
        <Stepper
          label="Maximum"
          value={draft.ageMax}
          min={draft.ageMin}
          max={99}
          onChange={(v) => set({ ageMax: v })}
          testID="age-max"
        />
        {errors.age ? <AppText variant="error">{errors.age}</AppText> : null}
      </View>
      <View style={{ gap: spacing.sm }}>
        <AppText variant="small" style={{ fontWeight: "600" }}>
          Distance maximale
        </AppText>
        <Row style={{ flexWrap: "wrap", gap: spacing.sm }}>
          {DISTANCES.map((d) => (
            <Chip
              key={d.value}
              testID={`distance-${d.value}`}
              label={d.label}
              selected={draft.maxDistanceKm === d.value}
              onPress={() => set({ maxDistanceKm: d.value })}
            />
          ))}
        </Row>
      </View>
    </View>
  );
}

function ToggleRow({
  label,
  hint,
  value,
  onChange,
  testID,
}: {
  label: string;
  hint: string;
  value: boolean;
  onChange: (v: boolean) => void;
  testID: string;
}) {
  return (
    <Row style={{ justifyContent: "space-between", alignItems: "flex-start" }}>
      <View style={{ flex: 1, gap: 2 }}>
        <AppText style={{ fontWeight: "600" }}>{label}</AppText>
        <AppText variant="muted">{hint}</AppText>
      </View>
      <Switch testID={testID} accessibilityLabel={label} value={value} onValueChange={onChange} />
    </Row>
  );
}

export function PrivacySection({ draft, set }: SectionProps) {
  return (
    <View style={{ gap: spacing.lg }}>
      <ToggleRow
        testID="show-distance"
        label="Afficher ma distance"
        hint="Les autres voient une distance approximative, jamais ma position."
        value={draft.showDistance}
        onChange={(v) => set({ showDistance: v })}
      />
      <ToggleRow
        testID="discoverable"
        label="Apparaître dans la découverte"
        hint="Désactivez pour mettre votre profil en pause. Vos matchs et conversations restent actifs."
        value={draft.discoverable}
        onChange={(v) => set({ discoverable: v })}
      />
    </View>
  );
}
