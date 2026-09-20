import UIKit
import Capacitor
import Security
import LocalAuthentication

@objc(SecureWalletVaultPlugin)
class SecureWalletVaultPlugin: CAPPlugin, CAPBridgedPlugin {
    let identifier = "SecureWalletVaultPlugin"
    let jsName = "SecureWalletVault"
    let pluginMethods: [CAPPluginMethod] = [
        CAPPluginMethod(name: "load", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "save", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "remove", returnType: CAPPluginReturnPromise)
    ]

    private let service = "com.agentwallet.app.secure-vault"
    private let account = "primary"

    @objc func load(_ call: CAPPluginCall) {
        var query = baseQuery()
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        if status == errSecItemNotFound {
            call.resolve(["value": NSNull()])
            return
        }
        guard status == errSecSuccess, let data = item as? Data,
              let value = String(data: data, encoding: .utf8) else {
            call.reject("Unable to load the encrypted wallet vault", nil, nil, ["status": status])
            return
        }
        call.resolve(["value": value])
    }

    @objc func save(_ call: CAPPluginCall) {
        guard let value = call.getString("value"), let data = value.data(using: .utf8) else {
            call.reject("Vault value is required")
            return
        }
        let query = baseQuery()
        let update = [kSecValueData as String: data]
        let updateStatus = SecItemUpdate(query as CFDictionary, update as CFDictionary)
        if updateStatus == errSecSuccess {
            call.resolve()
            return
        }
        guard updateStatus == errSecItemNotFound else {
            call.reject("Unable to update the encrypted wallet vault", nil, nil, ["status": updateStatus])
            return
        }
        var insert = query
        insert[kSecValueData as String] = data
        insert[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        let insertStatus = SecItemAdd(insert as CFDictionary, nil)
        guard insertStatus == errSecSuccess else {
            call.reject("Unable to save the encrypted wallet vault", nil, nil, ["status": insertStatus])
            return
        }
        call.resolve()
    }

    @objc func remove(_ call: CAPPluginCall) {
        let status = SecItemDelete(baseQuery() as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            call.reject("Unable to remove the encrypted wallet vault", nil, nil, ["status": status])
            return
        }
        call.resolve()
    }

    private func baseQuery() -> [String: Any] {
        return [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]
    }
}

@objc(WalletBiometricsPlugin)
class WalletBiometricsPlugin: CAPPlugin, CAPBridgedPlugin {
    let identifier = "WalletBiometricsPlugin"
    let jsName = "WalletBiometrics"
    let pluginMethods: [CAPPluginMethod] = [
        CAPPluginMethod(name: "status", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "saveCredential", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "authenticate", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "removeCredential", returnType: CAPPluginReturnPromise)
    ]

    private let service = "com.agentwallet.app.biometric-unlock"
    private let account = "primary"
    private let enabledKey = "agent-wallet-biometric-enabled"

    @objc func status(_ call: CAPPluginCall) {
        let context = LAContext()
        var error: NSError?
        let available = context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error)
        let type: String
        switch context.biometryType {
        case .faceID: type = "face"
        case .touchID: type = "fingerprint"
        case .opticID: type = "iris"
        default: type = "none"
        }
        call.resolve([
            "available": available,
            "enrolled": available,
            "enabled": UserDefaults.standard.bool(forKey: enabledKey),
            "biometryType": type,
            "label": type == "face" ? "Face ID" : type == "fingerprint" ? "Touch ID" : "生物识别"
        ])
    }

    @objc func saveCredential(_ call: CAPPluginCall) {
        guard let credential = call.getString("credential"), !credential.isEmpty,
              let data = credential.data(using: .utf8) else {
            call.reject("Credential is required")
            return
        }
        let context = LAContext()
        let reason = call.getString("reason") ?? "启用钱包快捷解锁"
        context.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, localizedReason: reason) { [weak self] success, error in
            guard let self else { return }
            guard success else {
                call.reject("Biometric authentication failed", nil, error)
                return
            }
            var accessError: Unmanaged<CFError>?
            guard let access = SecAccessControlCreateWithFlags(
                nil,
                kSecAttrAccessibleWhenPasscodeSetThisDeviceOnly,
                .biometryCurrentSet,
                &accessError
            ) else {
                call.reject("Unable to create biometric access control", nil, accessError?.takeRetainedValue())
                return
            }
            _ = SecItemDelete(self.baseQuery() as CFDictionary)
            var query = self.baseQuery()
            query[kSecValueData as String] = data
            query[kSecAttrAccessControl as String] = access
            let status = SecItemAdd(query as CFDictionary, nil)
            guard status == errSecSuccess else {
                call.reject("Unable to save biometric credential", nil, nil, ["status": status])
                return
            }
            UserDefaults.standard.set(true, forKey: self.enabledKey)
            call.resolve()
        }
    }

    @objc func authenticate(_ call: CAPPluginCall) {
        guard UserDefaults.standard.bool(forKey: enabledKey) else {
            call.reject("Biometric unlock is not enabled")
            return
        }
        let context = LAContext()
        context.localizedReason = call.getString("reason") ?? "验证身份以解锁钱包"
        var query = baseQuery()
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        query[kSecUseAuthenticationContext as String] = context
        DispatchQueue.global(qos: .userInitiated).async {
            var item: CFTypeRef?
            let status = SecItemCopyMatching(query as CFDictionary, &item)
            guard status == errSecSuccess, let data = item as? Data,
                  let credential = String(data: data, encoding: .utf8) else {
                if status == errSecItemNotFound {
                    UserDefaults.standard.removeObject(forKey: self.enabledKey)
                }
                call.reject("Biometric authentication failed", nil, nil, ["status": status])
                return
            }
            call.resolve(["credential": credential])
        }
    }

    @objc func removeCredential(_ call: CAPPluginCall) {
        let status = SecItemDelete(baseQuery() as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            call.reject("Unable to remove biometric credential", nil, nil, ["status": status])
            return
        }
        UserDefaults.standard.removeObject(forKey: enabledKey)
        call.resolve()
    }

    private func baseQuery() -> [String: Any] {
        return [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]
    }
}

class AgentWalletBridgeViewController: CAPBridgeViewController {
    override func capacitorDidLoad() {
        bridge?.registerPluginType(SecureWalletVaultPlugin.self)
        bridge?.registerPluginType(WalletBiometricsPlugin.self)
    }
}

@UIApplicationMain
class AppDelegate: UIResponder, UIApplicationDelegate {

    var window: UIWindow?

    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
        // Override point for customization after application launch.
        return true
    }

    func applicationWillResignActive(_ application: UIApplication) {
        // Sent when the application is about to move from active to inactive state. This can occur for certain types of temporary interruptions (such as an incoming phone call or SMS message) or when the user quits the application and it begins the transition to the background state.
        // Use this method to pause ongoing tasks, disable timers, and invalidate graphics rendering callbacks. Games should use this method to pause the game.
    }

    func applicationDidEnterBackground(_ application: UIApplication) {
        // Use this method to release shared resources, save user data, invalidate timers, and store enough application state information to restore your application to its current state in case it is terminated later.
        // If your application supports background execution, this method is called instead of applicationWillTerminate: when the user quits.
    }

    func applicationWillEnterForeground(_ application: UIApplication) {
        // Called as part of the transition from the background to the active state; here you can undo many of the changes made on entering the background.
    }

    func applicationDidBecomeActive(_ application: UIApplication) {
        // Restart any tasks that were paused (or not yet started) while the application was inactive. If the application was previously in the background, optionally refresh the user interface.
    }

    func applicationWillTerminate(_ application: UIApplication) {
        // Called when the application is about to terminate. Save data if appropriate. See also applicationDidEnterBackground:.
    }

    func application(_ application: UIApplication,
                     configurationForConnecting connectingSceneSession: UISceneSession,
                     options: UIScene.ConnectionOptions) -> UISceneConfiguration {
        let config = UISceneConfiguration(name: "Default Configuration",
                                          sessionRole: connectingSceneSession.role)
        config.delegateClass = SceneDelegate.self
        return config
    }
}
