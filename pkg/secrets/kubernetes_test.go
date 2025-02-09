package secrets

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

//type ExampleTestSuite struct {
//	suite.Suite
//}
//
//func (suite *ExampleTestSuite) SetupTest() {
//}
//func (suite *ExampleTestSuite) TestExample() {
//	assert.Equal(suite.T(), 5, 1)
//}
//
//func TestExampleTestSuite(t *testing.T) {
//	suite.Run(t, new(ExampleTestSuite))
//}

// Test:
//+------------------+-----+----------------+
//|                  | Pod | ServiceAccount |
//+------------------+-----+----------------+
//| ImagePullSecrets | Y   | Y              |
//+------------------+-----+----------------+
//| ImagePullSecrets | Y   | N              |
//+------------------+-----+----------------+
//| ImagePullSecrets | N   | Y              |
//+------------------+-----+----------------+
//| ImagePullSecrets | N   | N              |
//+------------------+-----+----------------+
//
// Multple image pull secrets on pod + service account
// Pod secret should override service account secret

func TestKubernetesCredentialProvider_GetImagePullSecrets(t *testing.T) {
	clientSet := fake.NewSimpleClientset()

	svcAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-service-account",
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "my-sa-secret"},
		},
	}
	svcAccountSecretDockerConfigJson := []byte(`{"auths":{"my-sa-secret.registry.example.com":{"username":"my-sa-secret","password":"xxxxxxxxxxx","email":"jdoe@example.com","auth":"c3R...zE2"}}}`)
	svcAccountSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-sa-secret",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: svcAccountSecretDockerConfigJson,
		},
	}
	podSecretDockerConfigJson := []byte(`{"auths":{"my-pod-secret.registry.example.com":{"username":"my-sa-secret","password":"xxxxxxxxxxx","email":"jdoe@example.com","auth":"c3R...zE2"}}}`)
	podSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-pod-secret",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: podSecretDockerConfigJson,
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "my-pod",
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "my-service-account",
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "my-pod-secret"},
			},
		},
	}

	_, _ = clientSet.CoreV1().ServiceAccounts("test-ns").Create(context.TODO(), svcAccount, metav1.CreateOptions{})
	_, _ = clientSet.CoreV1().Secrets("test-ns").Create(context.TODO(), svcAccountSecret, metav1.CreateOptions{})
	_, _ = clientSet.CoreV1().Secrets("test-ns").Create(context.TODO(), podSecret, metav1.CreateOptions{})

	provider := NewKubernetesImagePullSecretsProvider(clientSet)
	result, err := provider.GetImagePullSecrets(pod)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Secrets, 2)
	assert.Equal(t, svcAccountSecretDockerConfigJson, result.Secrets["my-sa-secret"])
	assert.Equal(t, podSecretDockerConfigJson, result.Secrets["my-pod-secret"])
}

// TestImagePullSecretsResult_Add tests if aggregation works
func TestImagePullSecretsResult_Add(t *testing.T) {
	expected := &ImagePullSecretsResult{
		Secrets: map[string][]byte{
			"foo": []byte("{\"foo\":\"123\"}"),
			"bar": []byte("{\"bar\":\"456\"}"),
		},
		Aggregate: []byte("{\"bar\":\"456\",\"foo\":\"123\"}"),
	}

	r := NewImagePullSecretsResult()
	r.Add("foo", []byte("{\"foo\":\"123\"}"))
	r.Add("bar", []byte("{\"bar\":\"456\"}"))

	assert.Equal(t, r, expected)
}
