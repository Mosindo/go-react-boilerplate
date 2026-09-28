import React from "react";
import type { MainScreenProps } from "../navigation/types";
import { ScreenContainer, Section } from "../shared/layout";
import { Text } from "../shared/ui";
import { BRAND } from "../theme";

export default function AboutScreen(_props: MainScreenProps<"About">) {
  return (
    <ScreenContainer scroll underHeader>
      <Text variant="title">{BRAND.name}</Text>
      <Text>
        {BRAND.name} is completely free. There are no ads, no subscriptions, no paywalled features and no limits on who
        you can like.
      </Text>
      <Section title="Your privacy">
        <Text>
          Your exact location is never shown. Coordinates are rounded to about 1 km, and other people only see an
          approximate distance if you allow it.
        </Text>
        <Text>Your birth date is private: only your age is displayed.</Text>
        <Text>
          Photos are only visible to people who are allowed to see your profile. You can pause your profile, block or
          report anyone, and delete your account and all of your data at any time.
        </Text>
      </Section>
      <Section title="Safety">
        <Text>
          Be kind. Report anything that feels wrong from a profile or a conversation. Blocked people disappear from your
          app immediately.
        </Text>
      </Section>
    </ScreenContainer>
  );
}
