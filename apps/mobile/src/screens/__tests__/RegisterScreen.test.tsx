import React from "react";
import { act, create, type ReactTestInstance } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import RegisterScreen from "../auth/RegisterScreen";
import * as authApi from "../../api/auth";

jest.mock("expo-secure-store", () => ({
  setItemAsync: jest.fn(),
  getItemAsync: jest.fn(async () => null),
  deleteItemAsync: jest.fn()
}));
jest.mock("../../auth/AuthContext", () => ({ useAuth: () => ({ signIn: jest.fn() }) }));
jest.mock("../../api/auth");

const IS_ACT_ENV = globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT?: boolean };
IS_ACT_ENV.IS_REACT_ACT_ENVIRONMENT = true;

type Navigation = React.ComponentProps<typeof RegisterScreen>["navigation"];
type Route = React.ComponentProps<typeof RegisterScreen>["route"];

function byTestId(root: ReactTestInstance, id: string): ReactTestInstance {
  // The first node carrying the testID together with a handler is the interactive component.
  const match = root.findAll(
    (n) => n.props.testID === id && (typeof n.props.onPress === "function" || typeof n.props.onChangeText === "function")
  )[0];
  if (!match) throw new Error(`no interactive element with testID ${id}`);
  return match;
}

async function render() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  const navigation = { navigate: jest.fn() } as unknown as Navigation;
  const route = { key: "r", name: "Register" } as Route;
  let tree!: ReturnType<typeof create>;
  await act(async () => {
    tree = create(
      <SafeAreaProvider
        initialMetrics={{ frame: { x: 0, y: 0, width: 400, height: 800 }, insets: { top: 0, left: 0, right: 0, bottom: 0 } }}
      >
        <QueryClientProvider client={client}>
          <RegisterScreen navigation={navigation} route={route} />
        </QueryClientProvider>
      </SafeAreaProvider>
    );
  });
  return tree;
}

function allText(tree: ReturnType<typeof create>): string {
  return JSON.stringify(tree.toJSON());
}

async function fill(tree: ReturnType<typeof create>, values: Record<string, string>) {
  for (const [id, value] of Object.entries(values)) {
    await act(async () => {
      byTestId(tree.root, id).props.onChangeText(value);
    });
  }
}

describe("RegisterScreen", () => {
  beforeEach(() => jest.clearAllMocks());

  it("blocks sign-up for under-18 users without calling the API", async () => {
    const tree = await render();
    const year = String(new Date().getFullYear() - 16);
    await fill(tree, {
      "register-email": "a@b.co",
      "register-password": "long-enough-pw",
      "register-day": "01",
      "register-month": "01",
      "register-year": year
    });
    await act(async () => {
      byTestId(tree.root, "register-submit").props.onPress();
    });
    expect(allText(tree)).toContain("majeures");
    expect(authApi.register).not.toHaveBeenCalled();
  });

  it("submits a valid adult registration with a normalized e-mail", async () => {
    (authApi.register as jest.Mock).mockResolvedValue({ accessToken: "a", refreshToken: "r", user: { id: "1" } });
    const tree = await render();
    await fill(tree, {
      "register-email": "  Someone@Example.COM ",
      "register-password": "long-enough-pw",
      "register-day": "12",
      "register-month": "04",
      "register-year": "1992"
    });
    await act(async () => {
      byTestId(tree.root, "register-submit").props.onPress();
    });
    expect(authApi.register).toHaveBeenCalledWith("someone@example.com", "long-enough-pw", "1992-04-12");
  });
});
