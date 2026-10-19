import {
  ageFromIso,
  formatBirthInput,
  formatDayHeader,
  formatListDate,
  parseBirthInput,
  toBirthInput
} from "../lib/dates";
import { ApiError, NetworkError } from "../api/client";
import { errorMessage } from "../lib/errors";
import { buildProfileInput } from "../components/ProfileForm";
import { distanceLabel } from "../components/SwipeCard";
import { insertMessage } from "../realtime/RealtimeProvider";
import { onboardingStep } from "../screens/OnboardingScreen";
import { notificationText } from "../screens/main/NotificationsScreen";
import { validateRegistration } from "../screens/auth/RegisterScreen";
import type { Message, MessagesPage, OwnProfile } from "../api/types";
import type { InfiniteData } from "@tanstack/react-query";

describe("birth date helpers", () => {
  it("formats digits as JJ/MM/AAAA while typing", () => {
    expect(formatBirthInput("1")).toBe("1");
    expect(formatBirthInput("120")).toBe("12/0");
    expect(formatBirthInput("12051990")).toBe("12/05/1990");
    expect(formatBirthInput("12/05/1990999")).toBe("12/05/1990");
  });

  it("parses only real calendar dates", () => {
    expect(parseBirthInput("12/05/1990")).toBe("1990-05-12");
    expect(parseBirthInput("29/02/2000")).toBe("2000-02-29");
    expect(parseBirthInput("29/02/2001")).toBeNull();
    expect(parseBirthInput("31/04/1990")).toBeNull();
    expect(parseBirthInput("1/5/1990")).toBeNull();
    expect(parseBirthInput("")).toBeNull();
  });

  it("round-trips to the display format", () => {
    expect(toBirthInput("1990-05-12")).toBe("12/05/1990");
  });

  it("computes age around the birthday", () => {
    const now = new Date(2026, 5, 15);
    expect(ageFromIso("2008-06-15", now)).toBe(18);
    expect(ageFromIso("2008-06-16", now)).toBe(17);
    expect(ageFromIso("1990-01-01", now)).toBe(36);
  });

  it("labels conversation dates", () => {
    const now = new Date(2026, 5, 15, 14, 0);
    expect(formatListDate(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toBe("09:05");
    expect(formatListDate(new Date(2026, 5, 14, 23, 0).toISOString(), now)).toBe("Hier");
    expect(formatListDate(new Date(2026, 5, 1, 10, 0).toISOString(), now)).toBe("01/06");
    expect(formatDayHeader(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toBe("Aujourd'hui");
  });
});

describe("errorMessage", () => {
  it("maps API codes and network failures to French messages", () => {
    expect(errorMessage(new ApiError(409, "email_taken", "email already exists"))).toMatch(/déjà utilisée/);
    expect(errorMessage(new ApiError(401, "invalid_credentials", "x"))).toMatch(/incorrect/);
    expect(errorMessage(new NetworkError())).toMatch(/Connexion impossible/);
    expect(errorMessage(new ApiError(500, undefined, "boom"))).toMatch(/de notre côté/);
    expect(errorMessage(new ApiError(400, undefined, "bad"))).toBe("bad");
    expect(errorMessage("weird")).toMatch(/inattendue/);
  });
});

describe("profile form validation", () => {
  const base = {
    firstName: " Léa ",
    birth: "12/05/1994",
    gender: "woman" as const,
    bio: " hi ",
    city: " Lyon ",
    interests: [1, 2],
    coords: null
  };

  it("builds a trimmed payload", () => {
    const { input, errors } = buildProfileInput(base);
    expect(errors).toEqual({});
    expect(input).toEqual({
      firstName: "Léa",
      birthDate: "1994-05-12",
      gender: "woman",
      bio: "hi",
      city: "Lyon",
      interests: [1, 2]
    });
  });

  it("includes coordinates only when provided", () => {
    const { input } = buildProfileInput({ ...base, coords: { lat: 1, lng: 2 } });
    expect(input?.location).toEqual({ lat: 1, lng: 2 });
  });

  it("rejects minors, invalid dates, missing name or gender", () => {
    const year = new Date().getFullYear() - 10;
    expect(buildProfileInput({ ...base, birth: `01/01/${year}` }).errors.birth).toMatch(/18 ans/);
    expect(buildProfileInput({ ...base, birth: "31/02/1990" }).errors.birth).toMatch(/invalide/);
    expect(buildProfileInput({ ...base, firstName: "  " }).errors.firstName).toBeDefined();
    expect(buildProfileInput({ ...base, gender: null }).errors.gender).toBeDefined();
    expect(buildProfileInput({ ...base, bio: "x".repeat(501) }).errors.bio).toBeDefined();
  });
});

describe("registration validation", () => {
  it("checks email, password rules and adult confirmation", () => {
    expect(validateRegistration("nope", "Password1", "Password1", true)).toMatch(/e-mail/);
    expect(validateRegistration("a@b.co", "short", "short", true)).toMatch(/8 caractères/);
    expect(validateRegistration("a@b.co", "é".repeat(40), "é".repeat(40), true)).toMatch(/trop long/);
    expect(validateRegistration("a@b.co", "Password1", "Password2", true)).toMatch(/correspondent/);
    expect(validateRegistration("a@b.co", "Password1", "Password1", false)).toMatch(/18 ans/);
    expect(validateRegistration("a@b.co", "Password1", "Password1", true)).toBeNull();
  });
});

describe("realtime cache helpers", () => {
  const message = (id: string): Message => ({
    id,
    conversationId: "c",
    senderId: "u",
    body: id,
    createdAt: "2026-01-01T00:00:00Z"
  });
  const page = (...ids: string[]): InfiniteData<MessagesPage> => ({
    pages: [{ messages: ids.map(message), peerLastReadAt: "2026-01-01T00:00:00Z" }],
    pageParams: [undefined]
  });

  it("prepends a new message to the first page", () => {
    expect(insertMessage(page("a"), message("b"))?.pages[0]?.messages.map((m) => m.id)).toEqual(["b", "a"]);
  });

  it("ignores duplicates (REST response + websocket echo)", () => {
    const cache = page("a", "b");
    expect(insertMessage(cache, message("b"))).toBe(cache);
  });

  it("leaves an unloaded conversation alone", () => {
    expect(insertMessage(undefined, message("a"))).toBeUndefined();
  });
});

describe("misc view helpers", () => {
  it("labels distances coarsely", () => {
    expect(distanceLabel(undefined)).toBeNull();
    expect(distanceLabel(1)).toBe("À moins de 1 km");
    expect(distanceLabel(15)).toBe("À 15 km");
  });

  it("describes notifications", () => {
    expect(
      notificationText({
        id: "1",
        type: "new_match",
        actor: { userId: "u", firstName: "Léa" },
        createdAt: ""
      })
    ).toMatch(/matché avec Léa/);
    expect(notificationText({ id: "1", type: "new_message", createdAt: "" })).toMatch(/Quelqu'un/);
  });

  it("derives the onboarding step from server state", () => {
    const profile = { photos: [], preferences: null } as unknown as OwnProfile;
    expect(onboardingStep(null, false)).toBe(1);
    expect(onboardingStep(profile, false)).toBe(2);
    expect(onboardingStep({ ...profile, photos: [{ id: "p", position: 0 }] }, false)).toBe(2);
    expect(onboardingStep({ ...profile, photos: [{ id: "p", position: 0 }] }, true)).toBe(3);
  });
});
