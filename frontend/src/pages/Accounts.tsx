import Header from '../components/Layout/Header'
import AccountList from '../components/Accounts/AccountList'

export default function Accounts() {
  return (
    <div>
      <Header
        title="Comptes"
        subtitle="Gérez vos comptes d'investissement"
      />

      <div className="p-8">
        <AccountList />
      </div>
    </div>
  )
}
